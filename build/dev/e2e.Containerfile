FROM node:24.22.1-trixie@sha256:016f604a11c4c725f0a4c62e29ff11adab153d297bf97046c9bf2ddb9f1011d3

# Chromium + system deps for Playwright, baked into the image so e2e runs
# offline. The playwright version must match @playwright/test in
# frontend/package.json.
ARG PLAYWRIGHT_VERSION=1.64.1
ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright

RUN npx -y playwright@${PLAYWRIGHT_VERSION} install --with-deps chromium && \
    rm -rf /var/lib/apt/lists/* /root/.npm
