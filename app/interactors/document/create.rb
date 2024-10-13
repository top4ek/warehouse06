# frozen_string_literal: true

module Interactors
  module Document
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
      end
    end
  end
end
