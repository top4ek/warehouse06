# app/routes/base.rb
require 'roda' # Ensure Roda is required

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


    # Add the code_reloader plugin for development environment
    if ENV['RACK_ENV'] == 'development'
      plugin :code_reloader
      # Configure directories to watch if needed by the plugin.
      # By default, it might watch the directory of the app file and subdirectories.
      # Explicitly watch key directories:
      # also_reload 'app/models', 'app/services', 'app/interactors', 'app/routes', 'app/serializers', 'app/contracts', 'config'
      # Roda's code_reloader might be simpler, often just needs to be enabled.
      # It typically reloads files under the app directory and the main app file itself.
      # Let's start with just enabling it and see if it picks up changes in `app/`.
      # If specific configuration like `also_reload` or `dont_reload` is needed,
      # it can be added later.
      # Some versions might use: `plugin :code_reloader, watch_subdirs: true`

      # For Roda, common usage is to just enable it and it watches the app file's dir.
      # To watch additional directories, you might need to configure `self.opts[:code_reloader_watch_subdirs]`
      # or use `Dir.glob` with `also_reload`.
      # Let's try a common approach for Roda's built-in reloader:
      # It often reloads constants.
      # We may need to specify what to reload or how.
      # A simple `plugin :code_reloader` might be enough to get started.
      # The plugin might also need `require 'roda/plugins/code_reloader'` if not auto-loaded.
      
      # Let's assume Roda can autoload its plugins if they are part of its core.
      # If not, an explicit require might be needed at the top of the file.
    end

    # ... rest of the class (e.g., route block if this is a base app) ...
    # If this Base class is not run directly but subclassed by Web and Api,
    # the plugin might need to be enabled in those subclasses or in a way that
    # the running Roda application instance gets it.
    # Let's assume for now enabling it in Base is sufficient if Web/Api inherit its plugins.
  end
end
