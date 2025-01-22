# frozen_string_literal: true

module Nodes
  module Rebuild
    class << self
      include Dry::Monads[:result]
      def call(_request)
        RebuildDbService.new.call.size
        Success(status: 200, title: 'Title', body: "megachponk: #{request.path}")
      end
    end
  end
end
