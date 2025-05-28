# app/routes/base.rb
require 'roda' # Ensure Roda is required
# Removed: require 'roda/plugins/code_reloader'

module Routes
  class Base < Roda
    # Common plugins
    plugin :default_headers,
           'Content-Type' => 'text/html', # Default, can be overridden
           'Strict-Transport-Security' => 'max-age=63072000; includeSubDomains',
           'X-Frame-Options' => 'deny',
           'X-Content-Type-Options' => 'nosniff'
    plugin :render, engine: 'slim', views: 'app/views', template_opts: { default_encoding: 'UTF-8' }
    plugin :sinatra_helpers # If still used, otherwise can be removed if not needed

    # Existing plugins from the original file should be merged here if they are still needed.
    # For example, :json, :json_parser, :error_handler, :common_logger
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

    # Removed the code_reloader plugin block for development environment

    # ... rest of the class (e.g., route block if this is a base app) ...
    # If this Base class is not run directly but subclassed by Web and Api,
    # the plugin might need to be enabled in those subclasses or in a way that
    # the running Roda application instance gets it.
    # Let's assume for now enabling it in Base is sufficient if Web/Api inherit its plugins.
  end
end
