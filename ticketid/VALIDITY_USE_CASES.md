# Ticket Validity Use Cases & Design Options

## Problem Statement

The current ticket ID structure uses a single `EventDate` field (5 characters, YYYYMMDD format). This works well for single-day events but doesn't handle tickets with extended validity periods.

## Use Case Analysis

### Category 1: Single-Day Events (Current Design Works)

**Examples:**
- Soccer matches: Valid only on match day (Dec 25, 2025)
- Concerts: Valid only on concert night
- Theater shows: Specific date and time
- Conference day passes: Single day only
- Sporting events: Game day only

**Characteristics:**
- Fixed date and time
- Ticket expires after the event
- Clear event date known at ticket creation
- No flexibility needed

**Current Solution:** Works perfectly - EventDate = event date

---

### Category 2: Multi-Day Validity from Purchase (Needs Solution)

**Examples:**
- **Amusement park tickets**: "Valid for 7 days from first use"
- **Museum passes**: "Valid for 30 days from purchase"
- **Gym day passes**: "Valid for 1 day after activation"
- **Transportation passes**: "Valid for 3 days"
- **Attraction bundles**: "Visit 5 attractions within 14 days"

**Characteristics:**
- Validity starts from purchase or first use
- Fixed duration (days/weeks/months)
- No specific event date
- User chooses when to use within validity window

**Current Solution:** Doesn't work well
- Setting EventDate = purchase date is misleading
- No way to encode validity duration
- Need external system to track validity period

---

### Category 3: Multi-Day Events (Needs Solution)

**Examples:**
- **Music festivals**: 3-day festival (Aug 15-17)
- **Conference passes**: 5-day conference
- **Season passes**: Valid entire season (3 months)
- **Annual passes**: Valid for 365 days from purchase
- **Weekend passes**: Valid Sat-Sun

**Characteristics:**
- Spans multiple consecutive or non-consecutive days
- May have start and end dates
- May be renewable
- Long validity periods

**Current Solution:** Partially works
- Could use EventDate = start date
- But no end date encoded
- No way to distinguish from single-day event

---

### Category 4: Flexible/Recurring Validity (Complex)

**Examples:**
- **10-visit punch cards**: Valid for 10 visits within 6 months
- **Season tickets**: Valid for all home games (variable dates)
- **Membership cards**: Monthly/yearly memberships
- **Class packages**: 20 classes, use anytime

**Characteristics:**
- Not date-based, usage-based
- Multiple uses
- Complex validity rules

**Current Solution:** Out of scope for ticket ID
- Too complex to encode in ID
- Requires external system for validation

---

## Design Options

### Option 1: Add ValidityDays Component (Recommended)

**Structure Change:**
```
Current Proposed: [EEE][DDDDD][CC][SSSS][RRR][XX] = 19 chars
New Proposal:     [EEE][DDDDD][CC][SSSS][VV][XX] = 18 chars

EEE    - Event ID (3 chars, 32K events)
DDDDD  - Start/Event Date (5 chars, YYYYMMDD)
CC     - Category (2 chars, 1K categories)
SSSS   - Seat ID (4 chars, 1M seats)
VV     - Validity Days (2 chars, 1,024 days)
XX     - Checksum (2 chars)
```

**Validity Days Encoding (2 chars = 1,024 values):**
- `0` = Single-day event (expires on EventDate)
- `1` = Valid for 1 day from EventDate
- `7` = Valid for 7 days from EventDate
- `30` = Valid for 30 days from EventDate
- `90` = Valid for 90 days from EventDate
- `365` = Valid for 1 year from EventDate
- `1023` = Valid for ~2.8 years (max)

**Semantic:**
- EventDate = Start date or first valid date
- ValidityDays = Number of days ticket remains valid
- Expiry = EventDate + ValidityDays

**Examples:**
```
Soccer Match (Dec 25, 2025):
  EventDate: 2025-12-25
  ValidityDays: 0
  → Valid only on Dec 25, 2025

Amusement Park (7-day pass, purchased Dec 1):
  EventDate: 2025-12-01
  ValidityDays: 7
  → Valid Dec 1-7, 2025

Museum Annual Pass (purchased Jan 1):
  EventDate: 2025-01-01
  ValidityDays: 365
  → Valid Jan 1, 2025 - Dec 31, 2025
```

**Pros:**
- ✅ Handles both single-day and multi-day tickets
- ✅ Clear semantics (start + duration)
- ✅ 1,024 days covers 99% of use cases
- ✅ Only 1 char shorter than previous proposal (19→18)
- ✅ No sequence component needed (or reduced to 1 char?)

**Cons:**
- ❌ Removes or reduces Sequence component
- ❌ Breaking change from current design
- ❌ More complex validation logic

**Trade-off Question:** Do we need Sequence component?
- Current proposal: 3 chars (32K sequences)
- Could reduce to 1 char (32 sequences) + add 2-char Validity
- Or remove entirely if not needed

---

### Option 2: Encode Validity in Category

**Structure:** Keep proposed 19-char structure
```
[EEE][DDDDD][CC][SSSS][RRR][XX] = 19 chars
```

**Category Encoding:**
- Use Category to encode BOTH ticket type AND validity
- Example: `CC` = 1,024 possible values
  - `0-99`: Single-day tickets (various categories)
  - `100-199`: 7-day validity tickets (various categories)
  - `200-299`: 30-day validity tickets (various categories)
  - etc.

**Examples:**
```
Category Mapping:
  01 = VIP Single-Day
  02 = General Single-Day
  03 = Student Single-Day
  101 = VIP 7-Day Pass
  102 = General 7-Day Pass
  201 = VIP 30-Day Pass
```

**Pros:**
- ✅ No structure change needed
- ✅ Keeps all 19 characters
- ✅ Flexible category encoding
- ✅ Simpler to implement

**Cons:**
- ❌ Reduces category flexibility (need encoding scheme)
- ❌ Not explicit (need lookup table)
- ❌ Harder to understand ticket ID at a glance
- ❌ Limited validity period options (must predefine)

---

### Option 3: Use EventDate as Expiry Date

**Structure:** Keep proposed 19-char structure
```
[EEE][DDDDD][CC][SSSS][RRR][XX] = 19 chars
```

**Semantic Change:**
- EventDate = Expiry date (not start date)
- Validity period calculated externally: Expiry - PurchaseDate

**Examples:**
```
Soccer Match (Dec 25, 2025):
  EventDate: 2025-12-25
  → Valid until Dec 25, 2025 end-of-day

7-Day Park Pass (purchased Dec 1, expires Dec 7):
  EventDate: 2025-12-07
  → Valid until Dec 7, 2025
```

**Pros:**
- ✅ No structure change
- ✅ Clear expiry date in ticket
- ✅ Simple validation (check current date ≤ EventDate)

**Cons:**
- ❌ Confusing semantics (EventDate != actual event date)
- ❌ Need external system to track purchase date
- ❌ Can't determine validity period from ticket alone
- ❌ Misleading for single-day events

---

### Option 4: Dual-Mode EventDate (Semantic Only)

**Structure:** Keep proposed 19-char structure
```
[EEE][DDDDD][CC][SSSS][RRR][XX] = 19 chars
```

**Semantic:**
- For single-day events: EventDate = event date
- For multi-day tickets: EventDate = start of validity
- Validity period stored in external system (database)
- Category indicates which mode

**Examples:**
```
Soccer Match:
  EventID: EVT123
  EventDate: 2025-12-25
  Category: SPORTS-SINGLE-DAY
  → Database: Validity = 0 days

Amusement Park:
  EventID: PARK001
  EventDate: 2025-12-01
  Category: PARK-7DAY
  → Database: Validity = 7 days
```

**Pros:**
- ✅ No structure change
- ✅ Flexibility in validity rules
- ✅ Can update validity rules without changing tickets

**Cons:**
- ❌ Ticket ID doesn't contain complete information
- ❌ Requires database lookup for validation
- ❌ Not self-contained
- ❌ Category becomes overloaded with meaning

---

### Option 5: Replace Sequence with Validity

**Structure:**
```
[EEE][DDDDD][CC][SSSS][VVV][XX] = 19 chars

EEE    - Event ID (3 chars)
DDDDD  - Event Date (5 chars)
CC     - Category (2 chars)
SSSS   - Seat ID (4 chars)
VVV    - Validity Days (3 chars, 32,768 days = 89 years)
XX     - Checksum (2 chars)
```

**Trade-off:**
- Remove Sequence entirely
- Add 3-char Validity component
- 32,768 days = 89+ years of validity

**When Sequence is Needed:**
- Multiple tickets for same Event+Date+Category+Seat
- Bulk generation with identical attributes
- Collision prevention

**When Sequence is NOT Needed:**
- Seat ID is always unique per ticket
- Event+Date+Category+Seat combo is always unique
- External system tracks ticket numbers

**Pros:**
- ✅ Explicit validity in ticket ID
- ✅ Huge range (89 years)
- ✅ Self-contained ticket information
- ✅ Keep 19-character length

**Cons:**
- ❌ Lose sequence for collision prevention
- ❌ Must ensure Event+Date+Category+Seat uniqueness
- ❌ Can't generate multiple identical tickets

---

## Comparison Matrix

| Option | Explicit Validity | Self-Contained | No Structure Change | Keeps Sequence | Complexity |
|--------|------------------|----------------|---------------------|----------------|------------|
| **1. Add ValidityDays (2-char)** | ✅ Yes | ✅ Yes | ❌ No (18 chars) | ⚠️ Reduced | Medium |
| **2. Encode in Category** | ❌ No (lookup) | ⚠️ Partial | ✅ Yes | ✅ Yes | Low |
| **3. EventDate as Expiry** | ⚠️ Implicit | ❌ No | ✅ Yes | ✅ Yes | Low |
| **4. Dual-Mode Semantic** | ❌ No (DB lookup) | ❌ No | ✅ Yes | ✅ Yes | Medium |
| **5. Replace Sequence** | ✅ Yes | ✅ Yes | ✅ Yes | ❌ No | Medium |

---

## Recommendations

### For Most Use Cases: **Option 5 (Replace Sequence with Validity)**

**Rationale:**
- Ticket uniqueness usually guaranteed by: EventID + Date + Category + SeatID
- Sequence is nice-to-have but not essential if seat IDs are always unique
- Validity period is more valuable information than sequence number
- Keeps 19-character length as planned

**When to use Sequence:**
- If you generate multiple tickets with identical Event+Date+Category+Seat
- If SeatID is not always unique (e.g., general admission)
- If you need collision resistance for concurrent generation

**When to use Validity:**
- If tickets have different validity periods
- If you sell multi-day passes, annual passes, etc.
- If validity is a core business requirement

### For Maximum Flexibility: **Option 4 (Dual-Mode with Database)**

**Rationale:**
- Keep ticket ID simple and short
- Store complex validation rules in database
- Can change validity rules without reissuing tickets
- Works for both simple and complex use cases

**Trade-off:**
- Requires database lookup to validate
- Ticket ID is not self-contained

---

## Questions to Answer

Before deciding, consider:

1. **Do you need Sequence component?**
   - Are Event+Date+Category+Seat combinations always unique?
   - Do you ever generate multiple tickets with identical attributes?
   - Do you need collision resistance?

2. **How important is self-contained validation?**
   - Should ticket ID contain all info needed to validate?
   - Or is database lookup acceptable?

3. **What validity periods do you need?**
   - Single-day only?
   - Common periods (1, 7, 30, 365 days)?
   - Arbitrary periods (1-1000 days)?

4. **What's your typical use case mix?**
   - 90% single-day, 10% multi-day → Option 4
   - 50/50 mix → Option 5
   - Mostly multi-day → Option 1 or 5

5. **Do you use seat IDs?**
   - If all tickets have unique seat IDs → Sequence less important
   - If general admission (no seats) → Sequence more important

---

## Proposed Decision Framework

```
START
  │
  ├─ Need self-contained validation?
  │  ├─ YES → Continue
  │  └─ NO → Use Option 4 (Dual-Mode Semantic)
  │
  ├─ Need Sequence for uniqueness?
  │  ├─ YES → Use Option 1 (Add Validity, reduce Sequence)
  │  └─ NO → Use Option 5 (Replace Sequence with Validity)
  │
  ├─ Complex validity rules?
  │  ├─ YES → Use Option 4 (Database-driven)
  │  └─ NO → Use Option 5 (Fixed validity periods)
  │
  └─ Minimize changes?
       ├─ YES → Use Option 2 (Encode in Category)
       └─ NO → Use Option 5 (Replace Sequence)
```

---

## My Recommendation

**Use Option 5: Replace Sequence with 3-char Validity**

**Structure:**
```
[EEE][DDDDD][CC][SSSS][VVV][XX] = 19 chars
```

**Assumptions:**
- Event+Date+Category+Seat is unique per ticket
- Validity period is more valuable than sequence
- Self-contained validation is important
- 32,768 days (89 years) covers all use cases

**If you need both Sequence AND Validity:**
- Use Option 1: Reduce Sequence to 1-2 chars, add 2-char Validity
- Structure: `[EEE][DDDDD][CC][SSSS][VV][R][XX]` = 19 chars
  - VV = 1,024 days validity
  - R = 32 sequences

What's your preference? Let's discuss which fields are essential for your use cases.
