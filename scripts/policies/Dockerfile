FROM alpine 
WORKDIR /opa-backend
RUN apk add --update \
    curl \
    && rm -rf /var/cache/apk/*
RUN curl -L -o opa https://openpolicyagent.org/downloads/v0.33.1/opa_linux_amd64_static && chmod 755 ./opa
COPY . . 
EXPOSE 8181
CMD ./opa run --server code.rego data.json 
