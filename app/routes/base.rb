# frozen_string_literal: true

require 'roda'

module Routes
  class Base < Roda
    plugin :common_logger
    plugin :json
    plugin :json_parser
    plugin :error_handler do |e|
      payload = { type: e.class.to_s }
      case e
      when Contracts::ValidationError, Sequel::ValidationFailed
        response.status = 422
        payload[:errors] = e.errors
      when Sequel::NoMatchingRow
        response.status = 404
        payload[:error] = 'Not found'
      else
        # Consider logging e.inspect and e.backtrace in development only
        # or to a proper logger service.
        # For now, keeping the original behavior.
        puts '================================================'
        puts e.inspect
        puts '================================================'
        puts e.backtrace
        response.status = 500
        payload[:error] = 'O-oh!'
      end
      response.write payload.to_json
    end

    route do |r|
      r.on 'api' do
        r.run Routes::Api
      end

      r.run Routes::Web
    end
  end
end
