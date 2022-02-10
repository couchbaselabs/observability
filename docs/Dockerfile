FROM antora/antora:3.0.1
# Antora requires the input be a Git repo with at least one commit: https://gitlab.com/antora/antora/-/issues/284
# hadolint ignore=DL3018
RUN apk add --upgrade --no-cache git && yarn global add http-server onchange && mkdir -p /antora/docs \
    && git init /antora && touch /antora/.gitignore && git -C /antora add . && \
    git config --global user.name "ignore.me" && \
    git config --global user.email "ignore.me" && \
    git -C /antora/docs commit -m "ignore me"
WORKDIR /antora/