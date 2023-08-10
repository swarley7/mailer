# mailer

## Description
Sends emails using templates. Useful for simple, bulk mailouts.


## Installation

Use one of the binaries in `builds/`

OR

`go build mailer.go`


## Usage


Copy / Paste an email into a file to form the template; substitute values within the email file with template variables:

~~~
{{ .FirstName }}
{{ .LastName }}
{{ .Email }}
{{ .Subject }}
{{ .From }}
{{ .Url }}
{{ .Date }}
~~~

Send the email using:

~~~sh
mailer -username "fred.tester@example" -password "StrongPassword123" -subject "TEST" -sender "Fred Tester <fred.tester@example>" -f test.csv -t template.eml -host smtp.office365.com -port 587 -debug -url "https://example.com/phishlol"
~~~

### Help text

~~~
Usage of ./mailer_macOS_amd64:
  -debug
    	enable verbosity
  -delay int
    	Delay between sending emails in ms (rate limit requests)
  -f string
    	csv file containing list of 'first_name,last_name,email' with header row (check example.csv)
  -host string
    	the smtp host
  -password string
    	the smtp password
  -port int
    	the smtp port (default 25)
  -sender string
    	the from address (default "Test Man <test@example.com>")
  -ssl
    	Force use of SSL (typically on port 465)
  -subject string
    	the email subject line (default "Test mail")
  -t string
    	Email template file
  -url string
    	The shonky URL to direct users to
  -username string
    	the smtp username
~~~

**Note** EMAIL SUCKS. Make sure you test delivery to known addresses before attempting to send to unknown addresses; chances are the email won't arrive / will render incorrectly / is full of mistakes (especially where HTML templates are in use).

## Support

Gitlab issues

## Roadmap

* Fix mailer to reuse a single connection instead of making a new SMTP client for every email

## Contributing
pull requests welcome


## Authors and acknowledgment


Peter Hannay <3 (@kronicd)

## License

MIT?

