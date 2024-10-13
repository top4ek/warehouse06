# frozen_string_literal: true

module Interactors
  module Directory
    class Create
      class << self
        def call(request)
          payload = Array.wrap(request.params['user'] || request.params['users'])

          contracts = payload.map { |p| validate p }

          models = contracts.map do |c|
            user = User.new(c.to_h)
            user.save
            user
          end
          models.size > 1 ? models : models.first
        end

        private

        def validate(payload)
          contract = Contracts::User::Create.new.call(payload)
          raise Contracts::ValidationError, contract unless contract.success?

          contract
        end
      end
    end
  end
end
