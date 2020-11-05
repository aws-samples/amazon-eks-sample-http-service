# AWS Sample: Simple Go Web Server for Amazon EKS
## Overview

This repository contains a webserver written in Go. Its only function is to
return a web page showing some interesting data about the Kubernetes Pod and EC2
instance on which it runs, along with remote IP address information.

This server can be used to illustrate the differences in behavior when you
choose Instances versus IP addresses in a Network or Application Load Balancer's
Target Group type. It can also be used to show the impact of enabling the Proxy
v2 Protocol on Network Load Balancers.

The server listens on ports 8080 and 9080. Port 9080 requires the [Proxy v2
protocol supported by AWS Network Load
Balancers](https://docs.aws.amazon.com/elasticloadbalancing/latest/network/load-balancer-target-groups.html#proxy-protocol)
to provide client IP address information. Port 8080 is for use without the Proxy
protocol.

## Prerequisites

You'll need to create an IAM policy as follows. The policy only allows the webserver
to describe EC2 network interfaces:

```sh
POLICY_ARN=$(aws iam create-policy \
  --policy-name EKSLoadBalancerDemo \
  --policy-document file://policy.json \
  --query 'Policy.Arn' --output text)
```

Then, you'll need to create a Service Account in your EKS cluster. This service
is called `aws-lb-demo` in the `default` namespace. The podspec located in
`deployment.yaml` refers to this name and namespace. You can change these if you
like, but you'll need to make sure the podspec is also changed if you do.

```sh
eksctl create iamserviceaccount \
  --cluster $CLUSTER \
  --attach-policy-arn $POLICY_ARN \
  --namespace default \
  --name aws-lb-demo \
  --override-existing-serviceaccounts \
  --approve
```

## Other files

The `deployment.yaml` file has a couple of application deployment manifests in
it. `ingresses.yaml` contains some service and ingress manifests. Finally, the
`nlb-services.yaml` file defines some Load Balancer services that create Network
Load Balancers. It creates multiple Load Balancers via both the legacy
in-tree NLB controller and the AWS Load Balancer v2 controller.

## Warnings

This software is unsupported and not for production use. It is for demonstration
purposes only.

Use of this software may cause you to incur AWS charges for the resources
created. Charges are the sole responsibility of the customer. We encourage you to
destroy these resources after you have finished using them.
