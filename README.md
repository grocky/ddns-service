

## References

### https://medium.com/aws-activate-startup-blog/building-a-serverless-dynamic-dns-system-with-aws-a32256f0a1d8

Base of proof of concept

### https://www.terraform.io/docs/providers/aws/guides/serverless-with-aws-lambda-and-api-gateway.html

Setting up a lambda with terraform

### https://giancarlopetrini.com/terraform-lambda-apigateway/

Gave me a hint about the following error:

```shell
curl http://_ff5c13ee6052045e98d30570a20e010d.ddns.rockygray.com
curl: (6) Could not resolve host: _ff5c13ee6052045e98d30570a20e010d.ddns.rockygray.com
```

Requests to the custom domain need to be over https!

> TODO: figure out http -> https redirect...

