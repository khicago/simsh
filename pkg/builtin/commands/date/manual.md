---
name: date
synopsis: "date [-u] [+FORMAT]"
category: system
---

# date -- display date and time

## SYNOPSIS

    date [-u] [+FORMAT]

## DESCRIPTION

Display the current date and time. Without a format specifier, outputs the
full date in default format. With a `+FORMAT` argument, outputs according
to the specified format string.

## FLAGS

- `-u` -- Use UTC instead of local time.

## FORMAT SPECIFIERS

- `%Y` -- Four-digit year (e.g. 2026)
- `%m` -- Two-digit month (01-12)
- `%d` -- Two-digit day (01-31)
- `%H` -- Hour in 24-hour format (00-23)
- `%M` -- Minute (00-59)
- `%S` -- Second (00-59)
- `%F` -- Full date, equivalent to `%Y-%m-%d`
- `%T` -- Full time, equivalent to `%H:%M:%S`
- `%s` -- Unix timestamp (seconds since epoch)
- `%z` -- Timezone offset (e.g. -0700)
- `%Z` -- Timezone abbreviation (e.g. MST)
- `%%` -- Literal percent sign

## EXAMPLES

Default format:

    date

UTC time:

    date -u

ISO date:

    date +%F

Unix timestamp:

    date +%s

Custom format:

    date "+%Y/%m/%d %H:%M:%S"

## NOTES

- Requires the `compat:date` capability in the compatibility profile.
- Only the listed format specifiers are supported.

## SEE ALSO

env
