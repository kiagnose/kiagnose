FROM registry.access.redhat.com/ubi8/ubi-minimal:8.6-994

RUN microdnf install shadow-utils && \
    adduser --system --no-create-home -u 1001 vm-latency && \
    microdnf remove shadow-utils && \
    microdnf clean all

COPY ./bin/kubevirt-vm-latency /usr/bin

USER 1001

CMD kubevirt-vm-latency
