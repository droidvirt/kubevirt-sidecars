FROM alpine:latest

RUN echo "https://mirrors.aliyun.com/alpine/v3.9/main" > /etc/apk/repositories \
	&& echo "https://mirrors.aliyun.com/alpine/v3.9/community" >> /etc/apk/repositories \
	&& apk --no-cache add libc6-compat bash iptables libvirt-dev

COPY proxy-sidecar /proxy-sidecar

ENTRYPOINT [ "/proxy-sidecar" ]