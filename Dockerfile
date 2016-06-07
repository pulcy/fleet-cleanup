FROM alpine:3.4

ADD ./fleet-cleanup /bin/

ENTRYPOINT ["/bin/fleet-cleanup"]
