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
      # r.root do
      #   view('root')
      # end

      result = Nodes::Show.call(r) # Changed Documents::Show to Nodes::Show
      result in status:, body:, title:, data:, path:
      response.status = status
      @title = title
      @body = body
      if result.success?
        r.redirect "/#{path}" if "/#{path}" != request.path

        if data
          send_file body, filename: title
        else
          view('directory')
        end
      else
        r.redirect '/'
      end
    end
  end
end
