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
        result = Nodes::Regenerate.call(r)
        if result.success?
          response.status = result.value![:status]
          { message: result.value![:message], nodes_processed: result.value![:nodes_processed] }
        else
          response.status = result.failure[:status]
          { error: result.failure[:message] }
        end
      end

      r.on 'nodes' do
        r.get do
          nodes = if r.params['path']
                    search_path = "#{r.params['path']}%"
                    ::Node.where(Sequel.like(:path, search_path)).all
                  else
                    ::Node.all
                  end
          response.status = 200
          Serializers::Node.render(nodes)
        end

        r.get /(.+)/ do |full_path|
          node = ::Node.first(path: full_path)

          if node
            response.status = 200
            Serializers::Node.render(node)
          else
            response.status = 404
            { error: "Node not found" }
          end
        end
      end

      r.on 'search' do
        r.get do
          query = r.params['q']

          if query.nil? || query.strip.empty?
            response.status = 200
            nodes = []
          else
            search_term = "%#{query.strip}%"
            nodes = ::Node.where(Sequel.ilike(:name, search_term) | Sequel.ilike(:description, search_term)).all
            response.status = 200
          end
          Serializers::Node.render(nodes)
        end
      end
    end
  end
end
