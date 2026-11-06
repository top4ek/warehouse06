FROM node:26.12.1-trixie@sha256:97fa0f1b59035c6072721a88d55be1371c7cb1cc942b521e1cc5c72ec87655f4

# Chromium + system deps for Playwright, baked into the image so e2e runs
# offline. The playwright version must match @playwright/test in
# frontend/package.json.
ARG PLAYWRIGHT_VERSION=1.64.1
ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright

RUN npx -y playwright@${PLAYWRIGHT_VERSION} install --with-deps chromium && \
    rm -rf /var/lib/apt/lists/* /root/.npm
