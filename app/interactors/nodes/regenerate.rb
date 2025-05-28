# frozen_string_literal: true

module Nodes
  module Regenerate # Changed from Rebuild
    class << self
      include Dry::Monads[:result]
      def call(_request) # request might not be needed if not used
        count = RebuildDbService.new.call
        Success(status: 200, message: "Successfully regenerated database.", nodes_processed: count) # Return count
      rescue StandardError => e
        Failure(status: 500, message: "Error regenerating database: #{e.message}")
      end
    end
  end
end
