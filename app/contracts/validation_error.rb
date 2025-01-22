# frozen_string_literal: true

module Contracts
  class ValidationError < StandardError
    attr_reader :errors

    def initialize(contract)
      super
      @errors = contract.errors.to_h
    end
  end
end
