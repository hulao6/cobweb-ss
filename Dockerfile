FROM alpine:latest
RUN   apk add --no-cache  ca-certificates
WORKDIR /app
VOLUME [ "/app/pb_data/" ]
EXPOSE 10000

COPY cobweb /app/cobweb
# start PocketBase
ENTRYPOINT [ "/app/cobweb", "serve", "--http=0.0.0.0:10000"  ]
CMD []
