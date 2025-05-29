# frozen_string_literal: true

RSpec.shared_examples_for "a model with factory" do
  let(:model) { create(described_class.to_s.underscore.to_sym) }

  it { expect(model).to be_valid }
end
