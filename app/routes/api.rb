# frozen_string_literal: true

module Routes
  class Api < Base
    plugin :heartbeat, path: '/ping'
    plugin :default_headers,
           'Content-Type' => 'application/json',
           'Strict-Transport-Security' => 'max-age=63072000; includeSubDomains',
           'X-Frame-Options' => 'deny',
           'X-Content-Type-Options' => 'nosniff'

    route do |r|
      r.get 'regenerate' do
        { size: Nodes::Regenerate.call(r) }
      end
    end
  end
end
