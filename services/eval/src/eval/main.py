import argparse
import json
import urllib.request

from eval.gold import GOLD


def _ask(base: str, question: str, timeout: float, token: str) -> dict:
    payload = json.dumps({"question": question}).encode("utf-8")
    request = urllib.request.Request(
        base.rstrip("/") + "/v1/ask",
        data=payload,
        headers={
            "Content-Type": "application/json",
            "Accept": "text/event-stream",
            "Authorization": f"Bearer {token}",
        },
        method="POST",
    )
    result = {"plan": None, "evidence": None, "answer": None}
    with urllib.request.urlopen(request, timeout=timeout) as response:
        event = ""
        for raw in response:
            line = raw.decode("utf-8").rstrip("\n")
            if line.startswith("event:"):
                event = line[6:].strip()
            elif line.startswith("data:"):
                data = json.loads(line[5:].strip())
                if event == "plan":
                    result["plan"] = data
                elif event == "evidence":
                    result["evidence"] = data
                elif event == "answer.done":
                    result["answer"] = data
    return result


def _check(case: dict, response: dict) -> tuple[bool, dict]:
    plan = response.get("plan") or {}
    evidence = response.get("evidence") or {}
    answer = response.get("answer") or {}
    guard = answer.get("guard") or {}

    metrics = {
        "intent": plan.get("intent"),
        "facts": len(evidence.get("facts", [])),
        "contradictions": len(evidence.get("contradictions", [])),
        "gaps": len(evidence.get("gaps", [])),
        "experts": len(evidence.get("experts", [])),
        "numbers_checked": guard.get("numbersChecked", 0),
        "guard_violations": guard.get("violations", -1),
    }

    checks = [
        metrics["guard_violations"] == 0,
        response.get("plan") is not None,
        response.get("evidence") is not None,
        response.get("answer") is not None,
        metrics["facts"] >= case.get("min_facts", 0),
        metrics["contradictions"] >= case.get("min_contradictions", 0),
        metrics["gaps"] >= case.get("min_gaps", 0),
        metrics["experts"] >= case.get("min_experts", 0),
    ]
    return all(checks), metrics


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base", default="http://localhost:8080")
    parser.add_argument("--timeout", type=float, default=20.0)
    parser.add_argument("--token", default="demo-researcher")
    args = parser.parse_args()

    rows = []
    total_numbers = 0
    total_violations = 0
    passed = 0
    for case in GOLD:
        try:
            response = _ask(args.base, case["question"], args.timeout, args.token)
            ok, metrics = _check(case, response)
        except Exception as error:
            ok, metrics = False, {"error": str(error)}
        rows.append((case["id"], ok, metrics))
        if ok:
            passed += 1
        total_numbers += metrics.get("numbers_checked", 0)
        total_violations += max(metrics.get("guard_violations", 0), 0)

    print("| вопрос | интент | facts | contr | gaps | experts | guard | итог |")
    print("|---|---|---|---|---|---|---|---|")
    for case_id, ok, metrics in rows:
        if "error" in metrics:
            print(f"| {case_id} | ошибка: {metrics['error'][:40]} | | | | | | ❌ |")
            continue
        print(
            f"| {case_id} | {metrics['intent']} | {metrics['facts']} | {metrics['contradictions']} "
            f"| {metrics['gaps']} | {metrics['experts']} | {metrics['numbers_checked']}/{metrics['guard_violations']} "
            f"| {'✅' if ok else '❌'} |"
        )

    rate = 0.0 if total_numbers == 0 else total_violations / total_numbers
    print()
    print(f"passed: {passed}/{len(GOLD)}")
    print(f"numbers checked: {total_numbers}")
    print(f"hallucinated_numbers_rate: {rate:.3f}")

    if passed != len(GOLD):
        raise SystemExit(1)


if __name__ == "__main__":
    main()
