# frozen_string_literal: true

require 'roda'

# require_relative 'routes/base' # Removed by Zeitwerk integration
# require_relative 'routes/api' # Removed by Zeitwerk integration
# require_relative 'routes/web' # Removed by Zeitwerk integration

class Application < Routes::Base
  route do |r|
    r.on 'api' do
      r.run Routes::Api
    end

    r.run Routes::Web
  end
end
