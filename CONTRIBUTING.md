# Contributing to Auto Claude Code

Thank you for your interest in contributing to Auto Claude Code! We welcome contributions from the community and are pleased to have them.

## 🤝 Ways to Contribute

- **Bug Reports**: Found a bug? Please let us know!
- **Feature Requests**: Have an idea for improvement? We'd love to hear it!
- **Code Contributions**: Want to implement a feature or fix a bug? Great!
- **Documentation**: Help improve our documentation
- **Testing**: Help test new features and bug fixes

## 🐛 Reporting Bugs

Before submitting a bug report, please:

1. **Check existing issues** to see if the bug has already been reported
2. **Use the latest version** to see if the bug still exists
3. **Gather information** about your environment:
   - Windows version
   - WSL version and distribution
   - Go version (if building from source)
   - Claude Code version

### Bug Report Template

```markdown
**Describe the bug**
A clear and concise description of what the bug is.

**To Reproduce**
Steps to reproduce the behavior:
1. Go to '...'
2. Click on '....'
3. Scroll down to '....'
4. See error

**Expected behavior**
A clear and concise description of what you expected to happen.

**Environment**
- OS: [e.g. Windows 11]
- WSL Version: [e.g. WSL2]
- WSL Distribution: [e.g. Ubuntu 22.04]
- Auto Claude Code Version: [e.g. v1.0.0]

**Additional context**
Add any other context about the problem here.
```

## 💡 Suggesting Features

Feature requests are welcome! Please provide:

1. **Clear description** of the feature
2. **Use case** - why would this be useful?
3. **Implementation ideas** (if you have any)

## 🔨 Development Setup

### Prerequisites

- Go 1.21+
- Windows 10/11 with WSL2
- Git
- Claude Code installed in WSL

### Setting Up Development Environment

1. **Fork the repository**
   ```bash
   # Fork on GitHub, then clone your fork
   git clone https://github.com/YOUR-USERNAME/auto-claude-code.git
   cd auto-claude-code
   ```

2. **Set up upstream remote**
   ```bash
   git remote add upstream https://github.com/original-owner/auto-claude-code.git
   ```

3. **Install dependencies**
   ```bash
   go mod download
   ```

4. **Build the project**
   ```bash
   go build -o auto-claude-code.exe ./cmd/auto-claude-code
   ```

5. **Run tests**
   ```bash
   go test ./...
   ```

## 📝 Coding Standards

### Go Code Style

- Follow standard Go formatting (`gofmt`)
- Use meaningful variable and function names
- Add comments for exported functions and complex logic
- Write tests for new functionality

### Commit Messages

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
type(scope): description

[optional body]

[optional footer]
```

**Types:**
- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `style`: Changes that do not affect the meaning of the code
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `test`: Adding missing tests or correcting existing tests
- `chore`: Changes to the build process or auxiliary tools

**Examples:**
```
feat(wsl): add support for Alpine Linux distribution
fix(path): handle spaces in Windows directory names
docs: update installation instructions
```

## 🧪 Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for specific package
go test ./internal/wsl
```

### Writing Tests

- Write unit tests for new functions
- Include integration tests for significant features
- Test error conditions and edge cases
- Use table-driven tests where appropriate

## 📋 Pull Request Process

1. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes**
   - Write clean, readable code
   - Add tests for new functionality
   - Update documentation if needed

3. **Test your changes**
   ```bash
   go test ./...
   go build ./cmd/auto-claude-code
   ```

4. **Commit your changes**
   ```bash
   git add .
   git commit -m "feat: add your feature description"
   ```

5. **Push to your fork**
   ```bash
   git push origin feature/your-feature-name
   ```

6. **Create a Pull Request**
   - Go to GitHub and create a PR from your fork
   - Provide a clear description of your changes
   - Reference any related issues

### Pull Request Guidelines

- **Keep PRs focused** - one feature or fix per PR
- **Write descriptive titles** and descriptions
- **Include tests** for new functionality
- **Update documentation** as needed
- **Ensure CI passes** before requesting review

## 📚 Documentation

### Updating Documentation

- Update relevant `.md` files in the `docs/` directory
- Update the main `README.md` if needed
- Include code examples for new features
- Keep documentation clear and concise

### Documentation Style

- Use clear, simple language
- Include code examples where helpful
- Use proper markdown formatting
- Add screenshots for UI changes (if applicable)

## 🔍 Code Review Process

1. **Automated checks** must pass (linting, tests, build)
2. **Manual review** by project maintainers
3. **Address feedback** - make requested changes
4. **Final approval** and merge

## 🌍 Internationalization

Currently, the project supports:
- English (primary)
- Chinese (Simplified) - 中文文档

If you'd like to add support for additional languages:
1. Translate documentation files
2. Add localized error messages (if applicable)
3. Update the README to mention the new language support

## 📞 Getting Help

- **GitHub Issues**: For bugs and feature requests
- **GitHub Discussions**: For questions and community discussion
- **Documentation**: Check the `docs/` directory

## 🎉 Recognition

Contributors will be:
- Listed in the project's contributors section
- Mentioned in release notes for significant contributions
- Invited to be maintainers for sustained contributions

Thank you for contributing to Auto Claude Code! 🙏 