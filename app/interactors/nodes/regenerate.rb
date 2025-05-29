# frozen_string_literal: true

module Nodes
  module Regenerate
    class << self
      include Dry::Monads[:result]
      def call(_request)
        count = RebuildDbService.new.call
        Success(status: 200, message: "Successfully regenerated database.", nodes_processed: count)
      rescue StandardError => e
        Failure(status: 500, message: "Error regenerating database: #{e.message}")
      end
    end
  end
end
