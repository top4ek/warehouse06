# frozen_string_literal: true

require 'redcarpet'

module Nodes
  module Show
    class << self
      include Dry::Monads[:result]

      def call(request)
        doc = document(request.path)
        return Failure(status: 404) if doc&.id.nil?

        result = { status: 200, title: doc.name, data: true, path: doc.path }

        if doc&.description.nil?
          result[:body] = Pathname(Dir.pwd).join('storage', doc.path)
        else
          result[:data] = false
          renderer = Redcarpet::Render::HTML.new(filter_html: true)
          result[:body] = Redcarpet::Markdown.new(renderer,
                                                  autolink: true,
                                                  tables: true)
                                             .render(doc.description)
        end
        Success(result.compact)
      end

      private

      def document(request_path)
        base = request_path.split('/').reject(&:empty?)
        path = [base, base + ['README.md']].map { it.join('/') }
        Document.where(path:).first
      end
    end
  end
end
