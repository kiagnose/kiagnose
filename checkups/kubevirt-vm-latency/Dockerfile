FROM registry.access.redhat.com/ubi9/ubi-minimal:9.4-949

RUN microdnf install -y shadow-utils && \
    adduser --system --no-create-home -u 1001 vm-latency && \
    microdnf remove -y shadow-utils && \
    microdnf clean all

COPY ./bin/kubevirt-vm-latency /usr/bin

USER 1001

ENTRYPOINT ["kubevirt-vm-latency"]
