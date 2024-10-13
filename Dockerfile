FROM ruby:3.3.5-alpine3.20

EXPOSE 3000
WORKDIR /opt

ARG CONTAINER_GID=1000
ARG CONTAINER_UID=1000

RUN addgroup -g ${CONTAINER_GID} app && \
  adduser --uid ${CONTAINER_UID} --disabled-password --ingroup app --home /opt app && \
  chown -R app:app /opt && \
  chmod -R 0775 /opt

RUN apk add --no-cache --update git build-base ca-certificates less tzdata make libffi-dev libxml2 libxml2-dev libxslt-dev

USER app:app

ADD --chown=app:app Gemfile Gemfile.lock /opt/
RUN bundle install --retry=3 --jobs 20

CMD ["bundle", "exec", "puma"]
