FROM golang:1.21 as build
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.mod go.sum .
RUN  go mod download
COPY ./debug .
RUN go build -o /bin/app .

FROM scratch
COPY --from=build /bin/app /bin/app
EXPOSE 8000
ENTRYPOINT ["./bin/app"]