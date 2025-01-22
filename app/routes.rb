# frozen_string_literal: true

require 'roda'

require_relative 'routes/base'
require_relative 'routes/api'
require_relative 'routes/web'

class Application < Routes::Base
  route do |r|
    r.on 'api' do
      r.run Routes::Api
    end

    r.run Routes::Web
  end
end
