package uidgenerator

import k8srand "k8s.io/apimachinery/pkg/util/rand"

func New() uidGenerator {
	return uidGenerator{}
}

type uidGenerator struct{}

func (u uidGenerator) UID() string {
	const uidLength = 8
	return k8srand.String(uidLength)
}
