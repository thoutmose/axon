# 0028 — Daily needs are reference intakes, not diets

- **Status:** Superseded by [0039](0039-daily-needs-follow-a-bounded-goal.md) (a bounded goal, 2026-10-20)
- **Date:** 2026-10-13
- **Epic:** EP-88

## Context

The owner asked for the amount of macronutrients a person needs "to be healthy", from their activity, age, sex, and size, with a warning to check with a nutritionist. Until now Axon set no numbers at all: the Nutritionist agent (EP-44) must never set calorie targets, because numeric targets drive restriction and can harm people with, or at risk of, eating disorders. Health logs are special-category data (EP-18).

## Decision

Axon works out **reference intakes for keeping one's weight**, in code, never by a model:

- Energy: basal rate by Mifflin–St Jeor (the most accurate common equation for adults), times an activity factor (1.2 to 1.9). Without a sex, the middle of the two equations' constants.
- Protein: EFSA's 0.83 g per kg for adults, up to 1.6 g/kg with hard training (ISSN position stand), and at least 1.0 g/kg from 65 (ESPEN).
- Fat 30 % of energy (EFSA reference intake 20–35 %), carbohydrates the rest (45–60 %), fibre 25 g and water 2.0 to 2.5 L (EFSA adequate intakes).

Every figure says how it was worked out and cites its source. A warning to check with a registered dietitian or nutritionist is always shown. There are **no numbers before 18 or during pregnancy or breastfeeding** (needs change with growth and month by month), and **no weight-loss or gain target**: a deficit is a clinical decision. Going over a reference is said in words, never in red. The profile is health data under the `health_data` consent, exported with it and deleted when it is withdrawn. Agents may read the needs to plan meals that meet them (EP-90); they still never set targets of their own.

## Consequences

People see what a day's log means against a sensible reference, and why. Those who want to lose or gain weight are pointed to a professional rather than given a number. The equations are estimates (Mifflin–St Jeor is within 10 % for most adults); the "How is this worked out?" panel says so through its sources.
