# frozen_string_literal: true

module Routes
  class Base < Roda
    plugin :common_logger
    # plugin :environments
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
        puts '================================================'
        puts e.inspect
        puts '================================================'
        puts e.backtrace
        response.status = 500
        payload[:error] = 'O-oh!'
        # payload[:inspect] = e.inspect
        # payload[:backtrace] = e.backtrace
      end
      response.write payload.to_json
    end
  end
end
