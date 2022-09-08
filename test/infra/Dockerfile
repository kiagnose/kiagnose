FROM quay.io/centos/centos:stream9

COPY requirements.txt /

RUN \
    dnf -y install \
      python3-pip \
    && \
    dnf clean all \
    && \
    python3 -m pip install -r requirements.txt

WORKDIR /workspace/kiagnose

CMD ["pytest"]
