# AWS based announce mailing list

	aws sns publish --subject "Managed services make it easy" --topic-arn $TOPIC --message file://announce-text.md

This project provides a Web form for people to subscribe to the $TOPIC with their email address.

# Why?

The typical way of setting up a mailing list via:

* mailman
* mailchimp

Are expensive to setup and run. Why not leverage AWS's
[SNS](https://aws.amazon.com/sns/) building block with the **email protocol**?

Admin page: https://ap-southeast-1.console.aws.amazon.com/sns/v3/home?region=ap-southeast-1#/topic/$TOPIC

# Limitations

* Does not support HTML / Rich text
* From address is <no-reply@sns.amazonaws.com>
* No archives

# Preventing abuse

	$ curl -d "email=foo@example.com" -X POST https://subscribe.dabase.com/
	Forbidden - CSRF token invalid
