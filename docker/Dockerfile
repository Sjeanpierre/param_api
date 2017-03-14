FROM alpine:3.3

RUN apk add --no-cache mysql-client

ADD ./bin/param-api-latest /bin/param-api

CMD ["/bin/param-api"]

EXPOSE 8080
