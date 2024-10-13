# frozen_string_literal: true

require 'roda'

require_relative 'routes/base'
require_relative 'routes/api'

class Application < Routes::Base
  plugin :heartbeat, path: '/api/ping'
  plugin :render, engine: 'slim', views: 'app/views', template_opts: { default_encoding: 'UTF-8' }

  route do |r|
    r.on 'api' do
      r.run Routes::Api
    end

    r.root do
      response.status = 200
      view('index')
    end
  end
end
