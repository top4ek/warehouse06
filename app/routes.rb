# frozen_string_literal: true

require 'roda'

require_relative 'routes/base'
require_relative 'routes/users'

class Application < Routes::Base
  plugin :heartbeat, path: '/ping'

  route do |r|
    r.root do
      response.status = 405
      ''
    end

    r.on 'users' do
      r.run Routes::Users
    end
  end
end
