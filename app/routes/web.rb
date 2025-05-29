# frozen_string_literal: true

module Routes
  class Web < Base
    plugin :default_headers,
           'Content-Type' => 'text/html',
           'Strict-Transport-Security' => 'max-age=63072000; includeSubDomains',
           'X-Frame-Options' => 'deny',
           'X-Content-Type-Options' => 'nosniff'
    plugin :render, engine: 'slim', views: 'app/views', template_opts: { default_encoding: 'UTF-8' }
    plugin :sinatra_helpers

    route do |r|
      # Get the result from the interactor
      interactor_result = Nodes::Show.call(r) # This returns a Monad (Success/Failure)

      # Set response status from the interactor result in all cases
      # For success, interactor_result.value! gives the hash.
      # For failure, interactor_result.failure gives the hash.
      payload = interactor_result.value_or { |failure_payload| failure_payload }
      response.status = payload[:status]
      @title = payload[:title]
      @body = payload[:body] # For directory view, this is markdown; for error view, an error message.

      if interactor_result.success?
        success_data = interactor_result.value! # This is the hash from Success()

        # Handle redirects:
        # Only redirect if the resolved path is different from the requested path,
        # AND it's not a direct file request.
        # This helps ensure canonical URLs for directory views.
        if success_data[:path] && "/#{success_data[:path]}" != request.path && !success_data[:file_to_send]
          r.redirect "/#{success_data[:path]}"
        end

        if success_data[:file_to_send]
          # If it's a file, body contains the full path to the file
          send_file success_data[:body], filename: success_data[:title]
        else
          # If it's not a file to send, it's a directory/markdown view.
          # @body (already set from payload[:body]) should be the markdown content.
          # The view 'directory' will use @body.
          view('directory')
        end
      else # interactor_result.failure?
        # @title and @body are already set for the error view.
        # Ensure you have an 'app/views/error.slim' that can display @title and @body.
        view('error')
      end
    end
  end
end
