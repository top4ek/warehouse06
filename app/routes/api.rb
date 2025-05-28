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
                    # Ensure the path parameter is clean and prevent directory traversal issues if necessary,
                    # though Sequel typically handles this well with placeholders.
                    # We also want to list immediate children or all descendants.
                    # For now, let's list all descendants.
                    search_path = "#{r.params['path']}%"
                    ::Node.where(Sequel.like(:path, search_path)).all # Added ::Node to specify top-level Node
                  else
                    ::Node.all # Added ::Node to specify top-level Node
                  end
          response.status = 200
          Serializers::Node.render_as_json(nodes)
        end

        # Route for specific node GET /api/nodes/*
        r.get /(.+)/ do |full_path|
          node = ::Node.first(path: full_path) # Use ::Node to specify top-level Node

          if node
            response.status = 200
            Serializers::Node.render_as_json(node)
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
            response.status = 200 # As per plan, return 200 and empty array for empty query
            nodes = []
          else
            search_term = "%#{query.strip}%"
            # Use ::Node to specify top-level Node model
            nodes = ::Node.where(Sequel.ilike(:name, search_term) | Sequel.ilike(:description, search_term)).all
            response.status = 200
          end
          Serializers::Node.render_as_json(nodes)
        end
      end
    end
  end
end
