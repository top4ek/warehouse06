# frozen_string_literal: true

module Contracts
  module User
    class Create < Dry::Validation::Contract
      params do
        required(:email).filled(:string).value(max_size?: 50, min_size?: 5)
        required(:password).filled(:string).value(max_size?: 50, min_size?: 8)
        optional(:name).maybe(:str?, max_size?: 20)
      end
    end
  end
end
