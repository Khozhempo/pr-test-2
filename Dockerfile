FROM scratch
ADD ca-certificates.crt /etc/ssl/certs/
ADD pr-test-2 /
CMD ["/pr-test-2"]
#ENTRYPOINT ["/pr-test-2"]
