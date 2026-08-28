# Contributing to HiddenLayer Azure APIM Guardrails

Thank you for your interest in contributing to HiddenLayer Azure APIM Guardrails! We welcome contributions from the community to help improve and grow this project. By contributing, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md).

## Ways to Contribute

There are several ways you can contribute to this project:

1. **Reporting Bugs**: If you encounter a bug or issue with the project, please open a GitHub issue to report it. Please provide detailed steps to reproduce the issue if possible.

2. **Suggesting Enhancements**: Have an idea for a new feature or improvement? We'd love to hear it! Open an issue on GitHub and describe your enhancement suggestion.

3. **Submitting Pull Requests**: If you'd like to contribute code changes, fixes, or new features, please fork the repository, make your changes, and submit a pull request. Make sure to follow our [coding guidelines](CODING_GUIDELINES.md) and provide a clear description of your changes.

4. **Documentation Improvements**: Documentation is key to helping others understand and use this project effectively. If you find errors or areas for improvement in the documentation, feel free to submit pull requests to update it.

## Getting Started

To get started with contributing to HiddenLayer Azure APIM Guardrails, follow these steps:

1. Fork the repository on GitHub.
2. Clone your forked repository to your local machine.
   ```
   git clone https://github.com/your-username/hiddenlayer-azure-apim-guardrails.git
   ```
3. Create a new branch for your changes.
   ```
   git checkout -b my-feature-branch
   ```
4. Make your changes and commit them.
   ```
   git commit -m "Add new feature"
   ```
5. Push your changes to your forked repository.
   ```
   git push origin my-feature-branch
   ```
6. Open a pull request on the original repository.

## Releasing

Releases are cut by maintainers. The release pipeline (`.github/workflows/cli-release.yml`) does the heavy lifting; the maintainer contract is:

1. Push a SemVer tag `vX.Y.Z` pointing at a commit on `main`.
   ```
   git tag v1.2.3 && git push origin v1.2.3
   ```
   The pipeline enforces both the tag shape (SemVer, with hyphen-only prerelease suffixes such as `v1.2.3-rc1`) and that the tagged commit is reachable from `main`.

2. **Never tag a commit that predates the current release pipeline.** A tag push runs the workflow file as it exists at the tagged commit — that is, the old pipeline — not the current one on `main`.

3. The pipeline builds and signs everything into a **draft** release. Publication only happens after the `release` environment approval and the publish gate. Tags with a prerelease suffix (e.g. `-rc1`) publish flagged as prereleases.

4. If a run fails, use "Re-run failed jobs" only, and only while the release is still a draft. Never re-run after publication: immutable releases reject asset changes.

5. If a release attempt is abandoned, delete the draft release and document the tag as burned in the next release's notes. Tags are never reused.

See [docs/signing.md](docs/signing.md) for the full signing policy and the consumer verification recipe.

## Code of Conduct

Please note that all contributors are expected to adhere to the [Code of Conduct](CODE_OF_CONDUCT.md). By participating in this project, you agree to abide by its terms.

## Questions and Feedback

If you have any questions, feedback, or need further assistance, please don't hesitate to reach out to us by opening an issue on GitHub.

Thank you for contributing to HiddenLayer Azure APIM Guardrails!
