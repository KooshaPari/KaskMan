# CI/CD Workflows Documentation

This directory contains GitHub Actions workflows for the KaskManager R&D Platform.

## Workflows Overview

### 🔄 Main CI/CD Pipeline (`ci.yml`)
**Triggers:** Push to main/develop/go-implementation branches, PRs to main/develop

**Jobs:**
- **nodejs-ci**: Tests Node.js application on multiple versions (18, 20)
  - ESLint code quality checks
  - TypeScript type checking
  - Jest test execution with coverage
  - Application build verification
  - Coverage reporting to Codecov

- **go-ci**: Tests Go module in kodevibe-go directory
  - Go module verification and dependency checks
  - Static analysis with `go vet` and `staticcheck`
  - Race condition testing
  - Security scanning with gosec
  - Build verification

- **security**: Comprehensive security scanning
  - Trivy filesystem vulnerability scanning
  - npm audit for Node.js dependencies
  - SARIF report generation for GitHub Security tab

- **code-quality**: Code quality analysis
  - SonarCloud integration for detailed metrics
  - CodeQL analysis for security and quality issues

- **integration**: End-to-end integration testing
  - Application startup verification
  - Basic health check validation

- **deployment-check**: Production readiness validation
  - Production build verification
  - Package validation
  - Deployment artifact checks

### 🔍 Pull Request Checks (`pr-checks.yml`)
**Triggers:** PR events (opened, synchronize, reopened, ready_for_review)

**Features:**
- **Draft PR handling**: Skips checks for draft PRs
- **Parallel execution**: Fast feedback with concurrent jobs
- **Quality gates**: Enforces code quality standards
- **Security scanning**: Multiple security tools integration
- **Performance checks**: Basic performance validation
- **Comprehensive reporting**: PR validation summary

**Jobs:**
- **quick-checks**: Fast feedback (lint, commit message validation)
- **quality-gates**: Test coverage, dependency checks, bundle size
- **security-scan**: Semgrep, TruffleHog secret scanning
- **performance**: Application startup and basic performance tests
- **pr-validation**: Consolidated status reporting

### 📦 Dependency Updates (`dependency-update.yml`)
**Triggers:** Weekly schedule (Mondays 9 AM UTC), manual dispatch

**Features:**
- Automated npm and Go dependency updates
- Security vulnerability fixes
- Automated PR creation with detailed change summary
- Test execution before PR creation

### 🤖 Dependabot Configuration (`dependabot.yml`)
**Features:**
- Weekly dependency updates for npm, Go modules, and GitHub Actions
- Automatic PR creation with proper labeling
- Version update filtering for major releases
- Reviewer and assignee configuration

## Security Features

### 🔒 Integrated Security Tools
- **Trivy**: Filesystem vulnerability scanning
- **npm audit**: Node.js dependency vulnerability checking
- **gosec**: Go security analyzer
- **Semgrep**: Multi-language static analysis
- **TruffleHog**: Secret detection
- **CodeQL**: Security and quality analysis

### 📊 Reporting
- SARIF format results uploaded to GitHub Security tab
- Integration with GitHub's security features
- Detailed security summaries in PR checks

## Quality Gates

### ✅ Required Checks for PR Merge
1. **Code Quality**
   - ESLint passes with no errors
   - TypeScript compilation (when applicable)
   - Go static analysis passes

2. **Testing**
   - All unit tests pass
   - Test coverage ≥ 70%
   - Integration tests pass

3. **Security**
   - No high-severity vulnerabilities
   - Secret scanning passes
   - Security static analysis passes

4. **Build**
   - Application builds successfully
   - No build errors or warnings

### ⚠️ Quality Warnings
- Performance degradation alerts
- Bundle size increase notifications
- New TODO/FIXME comments
- License compatibility issues

## Configuration Files

### `sonar-project.properties`
SonarCloud configuration for detailed code quality metrics:
- Source and test path definitions
- Coverage report configuration
- Quality gate settings
- Language-specific settings

### Pull Request Template
Standardized PR template ensuring:
- Proper change categorization
- Security consideration documentation
- Testing verification
- Breaking change documentation

## Usage Examples

### Running Workflows Locally
```bash
# Install Act for local workflow testing
npm install -g @nektos/act

# Test the main CI workflow
act push -W .github/workflows/ci.yml

# Test PR checks
act pull_request -W .github/workflows/pr-checks.yml
```

### Manual Workflow Triggers
```bash
# Trigger dependency updates manually
gh workflow run dependency-update.yml
```

### Monitoring and Debugging
- Check workflow status: GitHub Actions tab
- View security reports: Security tab → Code scanning alerts
- Monitor code quality: SonarCloud dashboard
- Track dependencies: Insights → Dependency graph

## Secrets Configuration

Required secrets for full functionality:
- `SONAR_TOKEN`: SonarCloud authentication
- `CODECOV_TOKEN`: Codecov integration (optional)

## Workflow Optimization

### Performance Optimizations
- **Caching**: Node modules and Go dependencies cached
- **Parallel execution**: Independent jobs run concurrently
- **Conditional execution**: Skip unnecessary jobs for draft PRs
- **Matrix testing**: Efficient multi-version testing

### Cost Optimization
- **Conditional triggers**: Avoid unnecessary runs
- **Job dependencies**: Sequential execution where needed
- **Timeout settings**: Prevent hung jobs
- **Resource optimization**: Appropriate runner sizing

## Troubleshooting

### Common Issues
1. **Workflow failures**: Check job logs in Actions tab
2. **Security scan false positives**: Review SARIF reports
3. **Test failures**: Ensure local tests pass before pushing
4. **Build failures**: Verify dependencies and build scripts

### Best Practices
- Keep workflows updated with latest action versions
- Monitor workflow run times and optimize as needed
- Regular review of security scan results
- Maintain up-to-date documentation

## Contributing

When modifying workflows:
1. Test changes locally with Act when possible
2. Validate YAML syntax before committing
3. Update this documentation for significant changes
4. Follow conventional commit format for workflow changes