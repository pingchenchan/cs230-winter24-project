FROM golang:1.22 as build
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.mod go.sum .
RUN  go mod download
COPY . .
RUN go build -o /bin/gtw .

FROM scratch
COPY --from=build /bin/gtw /bin/gtw
EXPOSE 3000
EXPOSE 5000
ENTRYPOINT ["./bin/gtw"]