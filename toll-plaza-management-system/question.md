
# Highway Toll Plaza Management System

You are tasked with building an in-memory backend for a highway toll operator.

You may use any mainstream programming language. No user interface, HTTP APIs, or database is required—only a library with tests. The primary focus is on working code, and the collection report must be demonstrably correct using sample data.

## Problem Statement

A highway consists of multiple toll plazas, each operating several toll booths (lanes).

Whenever a vehicle passes through a booth, a toll event must be recorded. The toll amount depends on the vehicle type and the direction of travel. The same vehicle type is charged differently for UP and DOWN journeys.

At the end of each day, the operator needs to know the total amount collected by each toll booth.

## Minimum Requirements

1. **Record Toll Events:** Record toll events as they occur.
2. **Maintain Rate Card:** Maintain a rate card defining the toll charges.
3. **Daily Collection Report:** Generate a daily collection report for each booth based on a given date.

## Future Considerations

The operator expects the pricing rules to evolve over time, potentially introducing:

- Festival discounts.
- Monthly vehicle passes.
- Annual vehicle passes.

You are not required to implement these features now. However, the pricing design should be flexible enough to accommodate new pricing rules without rewriting the existing implementation.

## Additional Instructions

Include tests to verify the implementation.

Clearly state all assumptions. If any requirement is unspecified, make a reasonable assumption and proceed. I will provide corrections if necessary.
