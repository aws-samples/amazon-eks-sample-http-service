package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pires/go-proxyproto"
	"github.com/sebest/xff"
)

const (
	docroot = "html"
)

type handler struct {
	Identity             ec2metadata.EC2InstanceIdentityDocument
	PodName              string
	PodNamespace         string
	AppLabel             string
	ProxyProtocolEnabled bool
}

type requestInfo struct {
	*handler
	URL        *url.URL
	PeerAddr   string
	RemoteAddr string
	PeerType   string
	RemoteType string
	ServerPort string
}

func maybeServeLocalFile(w http.ResponseWriter, r *http.Request) (bool, error) {
	path := docroot + r.URL.Path
	stat, err := os.Stat(path)
	if err != nil {
		if patherr, ok := err.(*os.PathError); ok && os.IsNotExist(patherr) {
			return false, nil
		}
		return false, err
	}
	if stat.IsDir() {
		return false, nil
	}
	f, err := os.Open(path)
	defer f.Close()

	if err != nil {
		return false, err
	}
	_, err = io.Copy(w, f)
	return true, err
}

func getRemoteType(region, addr string) (string, error) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))
	client := ec2.New(sess)
	output, err := client.DescribeNetworkInterfaces(&ec2.DescribeNetworkInterfacesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("addresses.private-ip-address"),
				Values: aws.StringSlice([]string{addr}),
			},
		},
	})
	if err != nil {
		return "", err
	}
	if len(output.NetworkInterfaces) == 0 {
		return "Non-VPC IP", nil
	}
	switch aws.StringValue(output.NetworkInterfaces[0].InterfaceType) {
	case "interface":
		instanceID := aws.StringValue(output.NetworkInterfaces[0].Attachment.InstanceId)
		if len(instanceID) >= 2 && instanceID[0:2] == "i-" {
			return "Instance ID " + instanceID, nil
		}
		description := aws.StringValue(output.NetworkInterfaces[0].Description)
		if len(description) >= 7 && description[0:7] == "ELB app" {
			return "Application load balancer", nil
		}
	case "network_load_balancer":
		return "Network Load Balancer", nil
	}
	return "Unknown", nil
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Requested: %v\n", r.URL)
	done, err := maybeServeLocalFile(w, r)
	if err != nil {
		fmt.Println(err)
		return
	}
	if done {
		return
	}

	peerAddr, _, _ := net.SplitHostPort(r.RemoteAddr)
	peerType, err := getRemoteType(h.Identity.Region, peerAddr)
	if err != nil {
		fmt.Println(err)
		return
	}

	remoteAddr, _, _ := net.SplitHostPort(xff.GetRemoteAddr(r))
	remoteType, err := getRemoteType(h.Identity.Region, remoteAddr)
	if err != nil {
		fmt.Println(err)
		return
	}

	i := requestInfo{
		handler:    h,
		URL:        r.URL,
		PeerAddr:   peerAddr,
		PeerType:   peerType,
		RemoteAddr: remoteAddr,
		RemoteType: remoteType,
	}

	r.Header.Get("X-Forwarded-For")

	localAddr := r.Context().Value(http.LocalAddrContextKey).(net.Addr).String()
	_, port, _ := net.SplitHostPort(localAddr)
	i.ServerPort = port

	t, _ := template.ParseFiles("html/index.html.tmpl")
	err = t.Execute(w, i)
	if err != nil {
		fmt.Println(err)
	}
}

func main() {
	sess := session.Must(session.NewSession())
	m := ec2metadata.New(sess)
	identity, err := m.GetInstanceIdentityDocument()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		h := &handler{
			Identity:     identity,
			PodName:      os.Getenv("POD_NAME"),
			PodNamespace: os.Getenv("POD_NAMESPACE"),
			AppLabel:     os.Getenv("APP_NAME"),
		}

		listenAddr := ":8080"

		mux := http.NewServeMux()
		mux.Handle("/", h)
		fmt.Printf("Listening on %s\n", listenAddr)
		log.Fatal(http.ListenAndServe(listenAddr, mux))
	}()
	go func() {
		h := &handler{
			Identity:             identity,
			PodName:              os.Getenv("POD_NAME"),
			PodNamespace:         os.Getenv("POD_NAMESPACE"),
			AppLabel:             os.Getenv("APP_NAME"),
			ProxyProtocolEnabled: true,
		}
		listenAddr := ":9080"

		l, err := net.Listen("tcp", listenAddr)
		if err != nil {
			log.Fatal(err)
		}

		proxyListener := &proxyproto.Listener{
			Listener: l,
			Policy: proxyproto.MustStrictWhiteListPolicy([]string{
				"10.0.0.0/8",
				"192.168.0.0/16",
			}),
		}
		defer proxyListener.Close()

		mux := http.NewServeMux()
		mux.Handle("/", h)
		fmt.Printf("Listening on %s\n", listenAddr)
		log.Fatal(http.Serve(proxyListener, mux))
	}()
	select {}
}
