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
        result = Interactors::Users::Create.call(r)
        response.status = 201
        if result.is_a? Array
          Serializers::User.render(result, root: :users, meta: { count: result.size })
        else
          Serializers::User.render(result, root: :user, meta: { link: result.to_link })
        end
      end
    end
  end
end
