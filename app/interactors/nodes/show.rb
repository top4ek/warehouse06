# frozen_string_literal: true

require 'redcarpet'

module Nodes
  module Show
    class << self
      include Dry::Monads[:result]

      def call(request)
        node_record = document(request.path)
        return Failure(status: 404) if node_record&.id.nil?

        result = { status: 200, title: node_record.name, data: true, path: node_record.path }

        if node_record&.description.nil?
          result[:body] = Pathname(Dir.pwd).join('storage', node_record.path)
        else
          result[:data] = false
          renderer = Redcarpet::Render::HTML.new(filter_html: true)
          result[:body] = Redcarpet::Markdown.new(renderer,
                                                  autolink: true,
                                                  tables: true)
                                             .render(node_record.description)
        end
        Success(result.compact)
      end

      private

      def document(request_path)
        base = request_path.split('/').reject(&:empty?)
        path = [base, base + ['README.md']].map { it.join('/') }
        Node.where(path:).first
      end
    end
  end
end
