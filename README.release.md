# Creating a New Release

## Changelog

- [ ] Update the CHANGELOG file in the project root directory.
  Use `git log  --oneline v0.1.0..HEAD` to get the changes since the last tag.
  Pick the most important changes, especially user facing ones.

  The following format should be used when adding an entry:

```
## [X.Y.Z] - YYYY-MM-DD
### Breaking Changes
 - ...

### New Features
 - ...

### Bug Fixes
 - ...
```

## Tagging

- [ ] Tag new release in git.
```bash
# Make sure your local git repo is sync with upstream.
# The whole version string should log like `v0.1.0`.
# For the commit message use the following format: `kiagnose 0.1.0 release`.
git tag --sign v<version>
git push upstream --tags
```

- [ ] In case there is a need to remove a tag:
```bash
# Remove local tag
git tag -d <tag_name>

# Remove upstream tag
git push --delete upstream <tag_name>
```

## Container Images

- [ ] Verify that the container images with the proper label exist in the registry:
  - [kiagnose](https://quay.io/repository/kiagnose/kiagnose?tab=tags)
  - [kubevirt-vm-latency](https://quay.io/repository/kiagnose/kubevirt-vm-latency?tab=tags)
  - [echo-checkup](https://quay.io/repository/kiagnose/echo-checkup?tab=tags)

## GitHub Release

- [ ] Visit [Create a new release](https://github.com/kiagnose/kiagnose/releases/new).
- [ ] Make sure you are in `Release` tab.
- [ ] Choose the git tag just pushed.
- [ ] Set title with the following format: `Version 0.1.0 release`.
- [ ] The content should be copied from the `CHANGELOG` file.
- [ ] Add the container images references to the content.
- [ ] Click `Save draft` and seek for review.
- [ ] Click `Publish release` once approved.
