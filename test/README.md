# End to End Tests

This is the home of the Kiagnose E2E tests.

## Content
- Infrastructure specification: [Container image specification](./infra/Dockerfile).
  A container based workspace on which the E2E can be executed.
- Test library helpers: [libtest](./libtest).
- Tests, separated by subjects in folders and files.

## Testing Framework

The tests are based on the [pytest](https://pytest.org/) testing python
framework.

## Development Environment

The tests are running in a containerized environment.
For local development, the infra image needs to be built.

### Build Image
In order to build the image, one can use podman:
`podman build -f ./Dockerfile -t kiagnose-e2e-test .`

### Running the Tests
To run the tests, just execute:
`podman run -ti --rm --net=host -v $(pwd):/workspace/kiagnose:Z -v ${HOME}/.kube:/root/.kube:ro,Z kiagnose-e2e-test`

> **Note**: The tests run on the host network namespace, where access to the k8s cluster is available.
> The `kubeconfig` configuration is shared through a volume to the container.

> **Note**: The default command execution runs all test.

### Running format & lint
The format and lint are processing only the python based test code.

- Format:
  `podman run -ti --rm -v $(pwd):/workspace/kiagnose:Z kiagnose-e2e-test black -S --check --diff ./test`
- Lint:
  `podman run -ti --rm -v $(pwd):/workspace/kiagnose:Z kiagnose-e2e-test python3 -m flake8 --max-line-length 100 ./test`


### Accessing the container (for debugging)
To access the shell in order to run individual commands, execute:
`podman run -ti --rm -v $(pwd):/workspace/kiagnose:Z kiagnose-e2e-test bash`
