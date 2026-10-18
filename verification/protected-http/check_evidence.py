#!/usr/bin/env python3
"""Check bounded HTTP CI provenance; this does not grade arbitrary implementations."""
from pathlib import Path
import hashlib
import base64
import json
import re
import sys

ROOT = Path(__file__).resolve().parents[2]
PACKAGE = "github.com/sage-x-project/sage-adk/core/a2abinding"
MODES = {"success", "repeat", "unsupported", "source-after-setup", "expired-after-setup", "cancel-after-setup", "close-after-setup", "source-after-handler", "expired-after-handler", "close-during-handler", "cancel-during-handler"}
GOOD = {"success", "repeat", "unsupported"}
CONTRACT = {
    "IngressRefusalEvidence": {"positive-calibration", "bare-jsonrpc", "legacy-unsigned", "grpc-media-http1", "h2-only"},
    "Runtime": {m + "/" + h for m in MODES for h in ("bare-completion", "http-completion")},
    "FixedFraming": {"truncated-head", "bare-newline", "duplicate-host", "duplicate-length", "conflicting-length", "folding", "invalid-name", "missing-colon", "missing-name", "control-value", "transfer-encoding", "trailer", "compression", "upgrade", "continue", "proxy", "forwarded", "x-forwarded", "keepalive", "leading-zero", "negative-length", "oversize-length", "nonnumeric-length", "empty-length", "missing-length", "short-body", "large-head", "request/method", "request/version", "request/host", "request/route", "request/absolute-target", "request/startline", "response/invalid-status", "response/unsupported-protocol", "response/nonterminal", "response/no-content", "response/outside-status"},
    "TransportConfiguration": {"plain", "userinfo", "fragment", "implicit-path", "uppercase", "default-port", "opaque", "space", "too-large", "nil", "skip", "connection-callback", "peer-callback", "config-callback", "keylog", "renegotiation", "random", "time", "minimum", "maximum", "h2"},
    "FinalAdmission": {"success", "nil-context", "canceled", "closed", "unavailable-source", "verify-error", "nil-card", "budget", "rollback", "card-expired", "card-key-expired", "wire-expired", "wire-future", "nil-writer", "empty-frame", "large-frame", "short-write", "write-error"},
    "ClockAndCardBounds": set(),
    "ShutdownDuringDial": {"canceled-dial", "late-connection"},
    "DispatchHook": {"once", "missing", "twice", "hook-error", "ignored-verify-error"},
    "DispatchBudgetIncludesVerify": set(),
    "MemoryTLS": {"policy-denial-reuse", "card-unavailable-after-handler"},
    "LocalIngressRefusals": {"consumed-session", "wrong-local", "already-serving", "closed", "closed-connection", "canceled-connection", "deadline-error", "handshake-eof", "empty-response", "unsigned-bad-request", "unsigned-unauthorized"},
    "ConstructionAndClosure": {"nil-session", "wrong-target", "ttl-zero", "ttl-large", "timeout-zero", "timeout-large", "tls-nil", "server-name", "bad-tls"},
}


INGRESS_CASES = {'positive-calibration', 'bare-jsonrpc', 'legacy-unsigned', 'grpc-media-http1', 'h2-only'}
INGRESS_TOP = 'TestProtectedHTTPIngressRefusalEvidence010/'
INGRESS_PEER = 'did:sage:web:agents.example.com:billing-bot'
INGRESS_CALLER = 'did:sage:web:agents.example.com:client'
INGRESS_TARGET = 'https://agents.example.com/a2a'


def canonical(value):
    return json.dumps(value,sort_keys=True,separators=(',',':')).encode()


def unsigned_frame(case):
    body = b'{"jsonrpc":"2.0","id":"one","method":"SendMessage","params":{"message":{"messageId":"memory","role":"ROLE_USER","parts":[{"text":"inert"}]}}}'
    media, extra = 'application/json', 'A2A-Version: 1.0\r\nA2A-Extensions: https://sage-x-project.github.io/sage-spec/a2a/0.10.0/extension\r\n'
    if case == 'legacy-unsigned':
        extra = ''
    if case == 'grpc-media-http1':
        media, body = 'application/grpc', b'inert unsigned unit bytes'
    return (f'POST /a2a HTTP/1.1\r\nHost: agents.example.com\r\nContent-Type: {media}\r\nContent-Length: {len(body)}\r\nConnection: close\r\n'+extra+'\r\n').encode()+body


def http_frame(raw):
    assert type(raw) is bytes and 0 < len(raw) <= 49152
    head, body = raw.split(b'\r\n\r\n',1)
    lines = head.decode('ascii').split('\r\n')
    headers = {}
    for line in lines[1:]:
        key,value = line.split(':',1);key=key.lower();value=value.strip()
        assert key not in headers
        headers[key]=value
    assert 'content-length' in headers and str(len(body)) == headers['content-length'] and 0<len(body)<=32768
    return lines[0],headers,body


def binary_url(text, size):
    assert type(text) is str and re.fullmatch('[A-Za-z0-9_-]+',text)
    raw=base64.urlsafe_b64decode(text+'='*(-len(text)%4))
    assert len(raw)==size and base64.urlsafe_b64encode(raw).rstrip(b'=').decode()==text
    return raw


def positive_envelope(headers, body, response):
    def closed(pairs):
        result=dict(pairs);assert len(result)==len(pairs);return result
    envelope=json.loads(body,object_pairs_hook=closed)
    base={'version','id','did','recipient','kid','created','expires','nonce','encoding','context_id','role','signature','session_id'}
    assert type(envelope) is dict and set(envelope)==base|({'data','message_id','request_hash','success'} if response else {'payload','metadata'})
    sender,recipient=(INGRESS_PEER,INGRESS_CALLER) if response else (INGRESS_CALLER,INGRESS_PEER)
    assert envelope['version']=='0.10.0' and envelope['did']==sender and envelope['recipient']==recipient and envelope['kid']==sender+'#sign' and envelope['role']==('responder' if response else 'initiator') and envelope['encoding']=='session'
    assert type(envelope['created']) is int and envelope['created']==100 and type(envelope['expires']) is int and envelope['expires']==130
    for key in ('id','context_id'):
        assert type(envelope[key]) is str and re.fullmatch('[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}',envelope[key])
    for key,size in [('nonce',16),('session_id',16),('signature',64)]:
        binary_url(envelope[key],size)
    ciphertext=envelope['data' if response else 'payload']
    assert type(ciphertext) is str and re.fullmatch('[A-Za-z0-9_-]+',ciphertext)
    binary_url(ciphertext,len(base64.urlsafe_b64decode(ciphertext+'='*(-len(ciphertext)%4))))
    assert 36<=len(base64.urlsafe_b64decode(ciphertext+'='*(-len(ciphertext)%4)))<=16384
    assert headers['content-type']=='application/json' and headers['x-sage-did']==sender and headers['x-sage-version']=='0.10.0'
    assert headers['content-digest']=='sha-256=:'+base64.b64encode(hashlib.sha256(body).digest()).decode()+':'
    components='"@method" "@target-uri" "@authority" "content-type" "content-digest" "x-sage-did" "x-sage-version"'
    if response:
        components='"@status" "@method";req "@target-uri";req "@authority";req "content-digest";req "signature";req "x-sage-version";req "content-type" "content-digest" "x-sage-did" "x-sage-version"'
    expected='sig1=('+components+');keyid='+json.dumps(envelope['kid'])+';alg="ed25519";created=100;expires=130;nonce='+json.dumps(envelope['nonce'])+';tag="sage-0.10.0"'
    assert headers['signature-input']==expected
    signature=headers['signature'];assert type(signature) is str and signature.startswith('sig1=:') and signature.endswith(':')
    raw=base64.b64decode(signature[6:-1],validate=True)
    assert len(raw)==64 and base64.b64encode(raw).decode()==signature[6:-1]
    if response:
        assert envelope['success'] is True
        binary_url(envelope['request_hash'],32)
    else:
        assert canonical(envelope['metadata'])==canonical({'a2a_version':'1.0','a2a_extensions':'https://sage-x-project.github.io/sage-spec/a2a/0.10.0/extension'})
    # Structural/correlation facts only. Cryptographic authentication and AEAD
    # plaintext belong to the actual core invocation recorded by the Go unit.
    return envelope


def check_ingress_witness(row):
    fields = {'case','transport','target','server_did','local_did','peer_did','source_did','source_version','source_service','tls_name','offered','negotiated','handshake_failed','handshake_error','input_hex','input_sha256','written','response_frame_hex','response_status','response_body_hex','authenticated_response','response_error','verified_payload_hex','verified_success','verified_code','handler_calls','handler_payloads_hex','accepted_requests','accepted_responses','parser_trace','core_trace','calibration'}
    assert type(row) is dict and set(row) == fields and row['case'] in INGRESS_CASES
    case=row['case'];good=case=='positive-calibration';h2=case=='h2-only'
    assert row['transport']=='net.Pipe/TLS' and row['tls_name']=='localhost' and row['target']==row['source_service']==INGRESS_TARGET
    assert row['server_did']==row['local_did']==row['source_did']==INGRESS_PEER and row['peer_did']==INGRESS_CALLER and row['source_version']=='3'
    assert all(type(row[k]) is int for k in ('written','response_status','handler_calls','accepted_requests','accepted_responses'))
    assert all(type(row[k]) is bool for k in ('handshake_failed','authenticated_response','verified_success'))
    assert all(type(row[k]) is str and re.fullmatch('(?:[0-9a-f]{2})*',row[k]) for k in ('input_hex','response_frame_hex','response_body_hex','verified_payload_hex'))
    raw=bytes.fromhex(row['input_hex'])
    assert row['input_sha256']==hashlib.sha256(raw).hexdigest() and row['written']==len(raw)
    assert row['offered']==(['h2'] if h2 else ['http/1.1']) and row['negotiated']==('' if h2 else 'http/1.1') and row['handshake_failed'] is h2
    assert row['handshake_error']==('remote error: tls: no application protocol' if h2 else '')
    assert row['parser_trace']==row['core_trace']==('NOT_REACHED' if h2 else 'NOT_OBSERVED')
    assert all(row[k]==int(good) for k in ('handler_calls','accepted_requests','accepted_responses'))
    payload=canonical({'jsonrpc':'2.0','id':'one','method':'GetTask','params':{'id':'task'}})
    assert row['handler_payloads_hex']==([payload.hex()] if good else []) and row['authenticated_response'] is good and row['verified_success'] is good
    assert row['verified_payload_hex']==(b'{"jsonrpc":"2.0","id":"one","result":{"ok":true}}'.hex() if good else '') and row['verified_code']==''
    assert row['response_error']==('' if good or h2 else 'authentication failed')
    if h2:
        assert raw==b'' and row['response_frame_hex']==row['response_body_hex']=='' and row['response_status']==0 and row['calibration'] is None
        return
    start,headers,body=http_frame(raw)
    assert start=='POST /a2a HTTP/1.1' and headers['host']=='agents.example.com'
    response=bytes.fromhex(row['response_frame_hex']);status,response_headers,response_body=http_frame(response)
    assert status.startswith('HTTP/1.1 '+str(row['response_status'])+' ') and response_body.hex()==row['response_body_hex']
    if good:
        assert row['response_status']==200 and row['calibration'] is None
        request_envelope=positive_envelope(headers,body,False)
        response_envelope=positive_envelope(response_headers,response_body,True)
        assert response_envelope['message_id']==request_envelope['id'] and response_envelope['request_hash']==base64.urlsafe_b64encode(hashlib.sha256(canonical(request_envelope)).digest()).rstrip(b'=').decode()
        assert response_envelope['context_id']==request_envelope['context_id'] and response_envelope['session_id']==request_envelope['session_id'] and response_envelope['id']!=request_envelope['id'] and response_envelope['nonce']!=request_envelope['nonce']
    else:
        assert raw==unsigned_frame(case) and row['response_status']==401 and response_body==b'authentication failed\n' and response_headers['content-type']=='text/plain' and not {'signature','signature-input'}&response_headers.keys()
        expected={'scope':'separate-fresh-memory-fixture','parser_error':'','body_hex':body.hex(),'open_error':'authentication failed','returned_request':False,'accepted_requests':0}
        assert canonical(row['calibration'])==canonical(expected)


def ingress_observations(rows):
    buffers,result={},{}
    for event in rows:
        if event.get('Action')!='output':
            continue
        test,output=event.get('Test',''),event.get('Output')
        assert type(test) is str and type(output) is str
        text=buffers.get(test,'')+output
        assert len(text)<=262144
        lines=text.split('\n');buffers[test]=lines.pop()
        for line in lines:
            if 'INGRESS_MEMORY_OBSERVATION ' not in line:
                continue
            assert test.startswith(INGRESS_TOP)
            case=test.removeprefix(INGRESS_TOP)
            def closed(pairs):
                value=dict(pairs);assert len(value)==len(pairs);return value
            witness=json.loads(line.split('INGRESS_MEMORY_OBSERVATION ',1)[1],object_pairs_hook=closed)
            assert case in INGRESS_CASES and witness['case']==case and case not in result
            check_ingress_witness(witness);result[case]=witness
    assert not any(text for test,text in buffers.items() if test.startswith(INGRESS_TOP) or 'INGRESS_MEMORY_OBSERVATION' in text)
    assert set(result)==INGRESS_CASES
    return result


def source_hashes(root=ROOT):
    paths = json.loads((root / "verification/protected-http/source-files.json").read_text())
    assert isinstance(paths, list) and len(paths) == len(set(paths)) and paths == sorted(paths)
    assert all(isinstance(p, str) and not Path(p).is_absolute() and ".." not in Path(p).parts for p in paths)
    return {p: hashlib.sha256((root / p).read_bytes()).hexdigest() for p in paths}


def check_modules(data, expected):
    decoder = json.JSONDecoder()
    selected = {}
    while data.strip():
        module, end = decoder.raw_decode(data.lstrip())
        data = data.lstrip()[end:]
        assert isinstance(module, dict) and module.get("Path") in expected and "Replace" not in module
        assert module["Path"] not in selected
        selected[module["Path"]] = {k: module[k] for k in ("Version", "Sum", "GoModSum")}
    assert selected == expected
    return selected


def events(path):
    rows = [json.loads(line) for line in path.read_text().splitlines()]
    assert rows and all(r.get("Package") == PACKAGE for r in rows)
    assert not any(r.get("Action") in {"skip", "fail"} for r in rows)
    assert sum(r.get("Action") == "pass" and "Test" not in r for r in rows) == 1
    return rows


def check_tests(focused, full):
    def passed(rows):
        values = [r["Test"] for r in rows if r.get("Action") == "pass" and "Test" in r]
        assert len(values) == len(set(values))
        return set(values)
    passed_full, passed_focus = passed(full), passed(focused)
    expected = set()
    for suite, children in CONTRACT.items():
        top = "TestProtectedHTTP" + suite + "010"
        expected.add(top)
        expected.update(top + "/" + child for child in children)
    assert passed_focus == expected and passed_focus <= passed_full
    # Markers supplement the actual harness PASS set, never replace it.
    markers = {}
    pattern = r"owned-production-http handshake=(true|false) mode=([a-z-]+) handler=(\d+) usable-result=(true|false)"
    for row in focused:
        match = re.search(pattern, row.get("Output", ""))
        if not match:
            continue
        handshake, mode, handler, usable = match.groups()
        branch = "http-completion" if handshake == "true" else "bare-completion"
        assert mode in MODES and row.get("Test") == "TestProtectedHTTPRuntime010/" + mode + "/" + branch
        key = (handshake, mode)
        assert key not in markers
        expected_handler = 2 if mode == "repeat" else int(mode in {"success", "source-after-handler", "expired-after-handler", "close-during-handler", "cancel-during-handler"})
        assert int(handler) == expected_handler and (usable == "true") == (mode in GOOD)
        markers[key] = int(handler)
    assert set(markers) == {(h, m) for h in ("true", "false") for m in MODES}
    leaves = {p for p in passed_focus if not any(q.startswith(p + "/") for q in passed_focus)}
    memory = ingress_observations(focused)
    full_memory = ingress_observations(full)
    return {"memory_ingress_controls": len(memory), "ingress_memory_observations": {"focused": memory, "full": full_memory}, "suites": len(CONTRACT), "leaf_tests": len(leaves), "owned_tls_scenarios": len(markers), "passed_tests": sorted(passed_focus)}


def coverage(path):
    result = {}
    for line in path.read_text().splitlines()[1:]:
        location, statements, count = line.split()
        name = location.rsplit(":", 1)[0].rsplit("/", 1)[-1]
        if name not in {"http_transport.go", "http_ingress.go"}:
            continue
        total, covered = result.get(name, (0, 0))
        n = int(statements)
        result[name] = (total + n, covered + (n if int(count) else 0))
    assert set(result) == {"http_transport.go", "http_ingress.go"}
    assert all(total and covered * 100 >= total * 90 for total, covered in result.values()), result
    return {k: {"statements": t, "covered": c} for k, (t, c) in result.items()}


def main(stage, directory):
    out = Path(directory)
    if stage == "sources":
        (out / "sources.json").write_text(json.dumps(source_hashes(), indent=2) + "\n")
    elif stage == "unchanged":
        assert json.loads((out / "sources.json").read_text()) == source_hashes()
    elif stage == "modules":
        expected = json.loads((ROOT / "verification/library-consumer/did-dependencies.json").read_text())
        check_modules((out / "modules.json").read_text(), expected)
    elif stage == "tests":
        report = check_tests(events(out / "tests.jsonl"), events(out / "all-tests.jsonl"))
        report["focused_coverage"] = coverage(out / "focused.cover")
        report["full_coverage"] = coverage(out / "all.cover")
        report["scope"] = "bounded owned TLS and unit integration, fixture authority; no operating Registry or historical Inspector closure"
        (out / "summary.json").write_text(json.dumps(report, indent=2) + "\n")
        print(json.dumps({k: v for k, v in report.items() if k != "passed_tests"}, indent=2))
    else:
        raise ValueError("unknown evidence stage")


if __name__ == "__main__":
    main(*sys.argv[1:])
