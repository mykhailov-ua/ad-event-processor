// Alloy model: activation cap (P-C5 sketch). Optional M3.2 gate.
// Run: alloy6 exec licensing_activation.als

abstract sig License {}
abstract sig Host {}

one sig ActivationPolicy {
  maxActivations: one Int
}

sig Activation {
  license: one License,
  host: one Host,
}

fact capPositive {
  ActivationPolicy.maxActivations > 0
}

pred withinCap {
  #Activation <= ActivationPolicy.maxActivations
}

run withinCap for 5

assert activationCap {
  all l: License | l -> Activation <= ActivationPolicy.maxActivations
}

check activationCap for 5
