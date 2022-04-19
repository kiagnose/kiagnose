FROM registry.access.redhat.com/ubi8/ubi-minimal:8.5-240.1648458092

RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" &&\
    chmod 744 kubectl &&\
    mv ./kubectl /usr/local/bin

COPY ./entrypoint /usr/local/bin

ENTRYPOINT entrypoint
