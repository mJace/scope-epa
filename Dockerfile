FROM alpine:3.3
COPY ./scope-volume-count /usr/bin/scope-volume-count
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
ENTRYPOINT ["/usr/bin/scope-volume-count"]