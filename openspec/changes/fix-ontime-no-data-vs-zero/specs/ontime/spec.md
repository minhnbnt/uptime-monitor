## Purpose

Computes on-time (uptime percentage) for a server over a requested time window or a calendar day, and reports clearly when no recorded state exists so that "no data" is never presented as "0% uptime".

## ADDED Requirements

### Requirement: Known status is exactly ON or OFF
The system SHALL treat a recorded status as known only when it is exactly `ON` or `OFF`. An empty or any other status value — including the empty string produced when no event matches a left join — SHALL be treated as unknown, and SHALL never be interpreted as a real status.

#### Scenario: Empty status from a missing boundary row
- **WHEN** a range query returns a boundary row with an empty status because no prior event exists
- **THEN** the empty status is treated as unknown, not as `OFF`

#### Scenario: Non-ON/OFF status value
- **WHEN** a status value other than `ON` or `OFF` is present in the event stream
- **THEN** the value is treated as unknown and contributes no on-time

### Requirement: Uptime result distinguishes no-data from 0%
For any requested window, the system SHALL report whether any known server state exists in that window (`has_data`). A window with no known state SHALL report `has_data: false` and SHALL NOT present a numeric uptime as if the server were down. A window that is fully offline but known SHALL report `has_data: true` with `uptime: 0`.

#### Scenario: No events in the window
- **WHEN** a window contains no known events at all
- **THEN** the result reports `has_data: false` and does not claim a `0%` uptime

#### Scenario: Fully offline but known
- **WHEN** the window is entirely covered by known `OFF` state
- **THEN** the result reports `has_data: true` with uptime `0`

#### Scenario: Fully online but known
- **WHEN** the window is entirely covered by known `ON` state
- **THEN** the result reports `has_data: true` with uptime `100`

### Requirement: Partial windows are flagged
When no known event precedes the start of the requested window, the system SHALL not assume the pre-window state. The observed window SHALL shrink to the first known event inside the window, the result SHALL be flagged `partial: true`, and the caller SHALL be able to determine where observed data begins.

#### Scenario: First known event is mid-window
- **WHEN** the first known event occurs after the requested window start
- **THEN** the result is flagged `partial: true`, observed data begins at that event, and uptime is computed only over the observed sub-window

#### Scenario: Known boundary precedes the window
- **WHEN** a known-status event exists before the requested window start
- **THEN** the window is not shrunk, the result is not flagged partial, and the boundary status seeds the start state

### Requirement: Calendar-day uptime uses the target day's window
The per-day uptime for a given day SHALL be computed over that day's own window — midnight to midnight for past days, and midnight to the current time for the current day. Computing one server's past day SHALL NOT reuse another day's or the current moment's window.

#### Scenario: Past day
- **WHEN** uptime is requested for a past calendar day
- **THEN** the window is that day from midnight to the following midnight

#### Scenario: Current day
- **WHEN** uptime is requested for the current calendar day
- **THEN** the window ends at the current time

### Requirement: No-data survives caching
The on-time cache SHALL store enough information to distinguish "no data yet" from any numeric uptime, including `0.00`. A cached value that is neither the no-data marker nor a parseable number SHALL be treated as a cache miss rather than guessed at.

#### Scenario: No-data value round-trips through cache
- **WHEN** a no-data result is written to the cache and later read back
- **THEN** it is restored as no-data, not as `0%`

#### Scenario: Corrupt cached value
- **WHEN** a cached value is neither the no-data marker nor a parseable number
- **THEN** the entry is treated as a cache miss and recomputed

### Requirement: API responses expose data-absence
Responses for both the range uptime endpoint and per-day server statistics SHALL carry a `has_data` flag, and the range response SHALL additionally carry a `partial` flag. Absent data SHALL be surfaced to API callers so a client can render "no data" instead of "0%".

#### Scenario: Range endpoint returns no-data
- **WHEN** a server has no recorded state in the requested range
- **THEN** the response reports `has_data: false` alongside the uptime value

#### Scenario: Per-day statistics carry has_data
- **WHEN** per-day server statistics are returned
- **THEN** each day carries `has_data` so the client can distinguish no-data days from genuinely `0%` days

### Requirement: gRPC per-day stats carry data-absence
The gRPC `GetServersOntime` per-day response SHALL carry a `has_data` flag on each day so the notification-service digest can distinguish "no data" from `0%`. A no-data day SHALL round-trip through the proto unchanged and SHALL NOT be reported as a numeric `0` to the consumer.

#### Scenario: No-data day over gRPC
- **WHEN** a day has no recorded state and the per-day stats are returned over gRPC
- **THEN** that day carries `has_data: false`, and the digest treats it as "no data" (renders the no-data marker) rather than `0%`

### Requirement: Interval merging preserves no-data buckets
When consecutive interval buckets are merged for presentation, a bucket with no data SHALL NOT be merged into a bucket reporting `0%` uptime, because the two mean different things.

#### Scenario: Adjacent no-data and 0% buckets
- **WHEN** a no-data interval is adjacent to a `0%` interval
- **THEN** the buckets are kept separate rather than merged
