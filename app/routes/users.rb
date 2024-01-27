# frozen_string_literal: true

module Routes
  class Users < Base
    route do |r|
      r.on Integer do |id|
        user = User.with_pk!(id)

        r.get do
          Serializers::User.render(user, root: :user, meta: { link: user.to_link })
        end
      end

      r.get do
        users = User.all
        Serializers::User.render(users, root: :users, meta: { count: users.size })
      end

      r.post do
        payload = r.params['user']

        contract = Contracts::User::Create.new.call(payload)
        raise Contracts::ValidationError.new(contract) unless contract.success?

        user = User.new(contract.to_h)
        user.save
        response.status = 201
        Serializers::User.render(user, root: :user, meta: { link: user.to_link })
      end
    end
  end
end
