FROM golang:1.21
WORKDIR /app
COPY . .
RUN go build -o pons ./cmd
CMD ["./pons", "start"]