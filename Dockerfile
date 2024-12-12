FROM golang:1.23.4-bookworm AS build

WORKDIR /app

# Install exiv2 v0.27.2 for https://github.com/kolesa-team/goexiv
RUN apt update && apt install -y libexiv2-dev

# Install gcc and make
RUN apt install -y make

# Install Templ and Go deps
RUN go install github.com/a-h/templ/cmd/templ@latest
COPY go.mod go.sum ./
RUN go mod download

# Copy over files
COPY Makefile tailwindcss tailwind.config.js ./
COPY main.go .
COPY src src

# Run make
RUN make gen

# Build variables
ARG CGO_ENABLED=1
ARG GOOS=linux

# Build binary
RUN go build -o home_api .


#FROM scratch AS release-stage
FROM debian:bookworm AS release-stage

RUN apt update && apt install -y libexiv2-dev
# COPY --from=build /usr/lib/x86_64-linux-gnu/libexiv2.so.27 /usr/lib/x86_64-linux-gnu
COPY --from=build /app/home_api .

CMD ["./home_api"]
