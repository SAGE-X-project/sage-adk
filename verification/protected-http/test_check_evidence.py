"""Evidence parser units use synthetic reports, never claim protocol execution."""
import contextlib
import io
import importlib.util
import hashlib
import base64
import json
from pathlib import Path
import tempfile
import unittest

SPEC = importlib.util.spec_from_file_location("protected_http_check", Path(__file__).with_name("check_evidence.py"))
check = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(check)


def mock_positive_frames():
    enc=lambda raw:base64.urlsafe_b64encode(raw).rstrip(b'=').decode()
    request={'version':'0.10.0','id':'00000000-0000-4000-8000-000000000001','did':check.INGRESS_CALLER,'recipient':check.INGRESS_PEER,'kid':check.INGRESS_CALLER+'#sign','created':100,'expires':130,'nonce':enc(bytes([1])*16),'encoding':'session','context_id':'00000000-0000-4000-8000-000000000003','role':'initiator','signature':enc(bytes([4])*64),'session_id':enc(bytes([3])*16),'payload':enc(bytes([6])*36),'metadata':{'a2a_version':'1.0','a2a_extensions':'https://sage-x-project.github.io/sage-spec/a2a/0.10.0/extension'}}
    response={k:v for k,v in request.items() if k not in ('payload','metadata')}
    response.update(id='00000000-0000-4000-8000-000000000002',did=check.INGRESS_PEER,recipient=check.INGRESS_CALLER,kid=check.INGRESS_PEER+'#sign',nonce=enc(bytes([2])*16),role='responder',signature=enc(bytes([5])*64),data=enc(bytes([7])*36),message_id=request['id'],request_hash=enc(hashlib.sha256(check.canonical(request)).digest()),success=True)
    def frame(value,response):
        body=check.canonical(value)
        components='"@method" "@target-uri" "@authority" "content-type" "content-digest" "x-sage-did" "x-sage-version"'
        if response:components='"@status" "@method";req "@target-uri";req "@authority";req "content-digest";req "signature";req "x-sage-version";req "content-type" "content-digest" "x-sage-did" "x-sage-version"'
        si='sig1=('+components+');keyid='+json.dumps(value['kid'])+';alg="ed25519";created=100;expires=130;nonce='+json.dumps(value['nonce'])+';tag="sage-0.10.0"'
        start='HTTP/1.1 200 OK\r\n' if response else 'POST /a2a HTTP/1.1\r\nHost: agents.example.com\r\n'
        return (start+'content-type: application/json\r\ncontent-digest: sha-256=:'+base64.b64encode(hashlib.sha256(body).digest()).decode()+':\r\nx-sage-did: '+value['did']+'\r\nx-sage-version: 0.10.0\r\nsignature-input: '+si+'\r\nsignature: sig1=:'+base64.b64encode(bytes([8])*64).decode()+':\r\nContent-Length: '+str(len(body))+'\r\nConnection: close\r\n\r\n').encode()+body
    return frame(request,False),frame(response,True),check.canonical(response)


def mock_ingress(case):
    # Synthetic framing/control data: signatures below are deliberately invalid
    # and never count as native or actual core/Verifier execution evidence.
    good=case=='positive-calibration';h2=case=='h2-only'
    request=(b'POST /a2a HTTP/1.1\r\nHost: agents.example.com\r\nContent-Type: application/json\r\nContent-Length: 2\r\nSignature: mock\r\nSignature-Input: mock\r\nConnection: close\r\n\r\n{}' if good else (b'' if h2 else check.unsigned_frame(case)))
    body=b'{}' if good else (b'' if h2 else b'authentication failed\n')
    response=b'' if h2 else ((b'HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nSignature: mock\r\nSignature-Input: mock\r\n' if good else b'HTTP/1.1 401 Unauthorized\r\nContent-Type: text/plain\r\n')+f'Content-Length: {len(body)}\r\nConnection: close\r\n\r\n'.encode()+body)
    if good:
        request,response,body=mock_positive_frames()
    payload=check.canonical({'jsonrpc':'2.0','id':'one','method':'GetTask','params':{'id':'task'}})
    calibration=None if good or h2 else {'scope':'separate-fresh-memory-fixture','parser_error':'','body_hex':request.split(b'\r\n\r\n',1)[1].hex(),'open_error':'authentication failed','returned_request':False,'accepted_requests':0}
    return {'case':case,'transport':'net.Pipe/TLS','target':check.INGRESS_TARGET,'server_did':check.INGRESS_PEER,'local_did':check.INGRESS_PEER,'peer_did':check.INGRESS_CALLER,'source_did':check.INGRESS_PEER,'source_version':'3','source_service':check.INGRESS_TARGET,'tls_name':'localhost','offered':['h2'] if h2 else ['http/1.1'],'negotiated':'' if h2 else 'http/1.1','handshake_failed':h2,'handshake_error':'remote error: tls: no application protocol' if h2 else '', 'input_hex':request.hex(),'input_sha256':hashlib.sha256(request).hexdigest(),'written':len(request),'response_frame_hex':response.hex(),'response_status':200 if good else (0 if h2 else 401),'response_body_hex':body.hex(),'authenticated_response':good,'response_error':'' if good or h2 else 'authentication failed','verified_payload_hex':b'{"jsonrpc":"2.0","id":"one","result":{"ok":true}}'.hex() if good else '', 'verified_success':good,'verified_code':'','handler_calls':int(good),'handler_payloads_hex':[payload.hex()] if good else [],'accepted_requests':int(good),'accepted_responses':int(good),'parser_trace':'NOT_REACHED' if h2 else 'NOT_OBSERVED','core_trace':'NOT_REACHED' if h2 else 'NOT_OBSERVED','calibration':calibration}


def good_rows():
    rows = []
    for suite, children in check.CONTRACT.items():
        top = "TestProtectedHTTP" + suite + "010"
        for name in [top] + [top + "/" + child for child in sorted(children)]:
            rows.extend([{"Package": check.PACKAGE, "Action": "run", "Test": name}, {"Package": check.PACKAGE, "Action": "pass", "Test": name}])
    for handshake in ("true", "false"):
        for mode in sorted(check.MODES):
            handler = 2 if mode == "repeat" else int(mode in {"success", "source-after-handler", "expired-after-handler", "close-during-handler", "cancel-during-handler"})
            rows.append({"Package": check.PACKAGE, "Action": "output", "Test": "TestProtectedHTTPRuntime010/" + mode + "/" + ("http-completion" if handshake == "true" else "bare-completion"), "Output": f"owned-production-http handshake={handshake} mode={mode} handler={handler} usable-result={str(mode in check.GOOD).lower()}"})
    for case in sorted(check.INGRESS_CASES):
        rows.append({"Package":check.PACKAGE,"Action":"output","Test":check.INGRESS_TOP+case,"Output":"INGRESS_MEMORY_OBSERVATION "+json.dumps(mock_ingress(case))+"\n"})
    rows.append({"Package": check.PACKAGE, "Action": "pass"})
    return rows


class EvidenceTests(unittest.TestCase):
    def test_complete_harness_and_marker_contract(self):
        rows = good_rows()
        result = check.check_tests(rows, rows)
        self.assertEqual((result["suites"], result["leaf_tests"], result["owned_tls_scenarios"]), (12, 135, 22))

    def test_missing_duplicate_or_false_observations(self):
        rows = good_rows()
        candidates = [rows[:-2], rows + [rows[1]], rows[2:]]
        changed = json.loads(json.dumps(rows))
        next(r for r in changed if "Output" in r)["Output"] = "owned-production-http handshake=true mode=success handler=0 usable-result=true"
        candidates.append(changed)
        candidates.append(rows + [next(r for r in rows if "Output" in r)])
        for candidate in candidates:
            with self.assertRaises(AssertionError):
                check.check_tests(candidate, rows)

    def test_runtime_marker_matches_actual_handshake_branch(self):
        rows = good_rows()
        changed = json.loads(json.dumps(rows))
        for row in changed:
            if "Output" in row:
                row["Output"] = row["Output"].replace("handshake=true", "handshake=SWAP").replace("handshake=false", "handshake=true").replace("handshake=SWAP", "handshake=false")
        self.assertEqual([r.get("Test") for r in changed], [r.get("Test") for r in rows])
        with self.assertRaises(AssertionError):
            check.check_tests(changed, rows)

    def test_event_fail_skip_wrong_subject(self):
        with tempfile.TemporaryDirectory() as directory:
            p = Path(directory, "events.jsonl")
            rows = good_rows()
            p.write_text("\n".join(json.dumps(r) for r in rows))
            self.assertEqual(check.events(p), rows)
            for extra in [{"Package": check.PACKAGE, "Action": "fail"}, {"Package": check.PACKAGE, "Action": "skip"}, {"Package": "different", "Action": "output"}]:
                p.write_text("\n".join(json.dumps(r) for r in rows + [extra]))
                with self.assertRaises(AssertionError):
                    check.events(p)

    def test_module_identity_no_replacement(self):
        expected = {"core": {"Version": "v1.2.3", "Sum": "h1:fixture", "GoModSum": "h1:mod"}}
        module = {"Path": "core", **expected["core"]}
        self.assertEqual(check.check_modules(json.dumps(module), expected), expected)
        for data in ["", json.dumps(module) * 2, json.dumps({**module, "Replace": {}}), json.dumps({**module, "Sum": "wrong"}), json.dumps({**module, "Path": "different"})]:
            with self.assertRaises(AssertionError):
                check.check_modules(data, expected)

    def test_sources_scope_and_coverage(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            manifest = root / "verification/protected-http/source-files.json"
            manifest.parent.mkdir(parents=True)
            (root / "source.go").write_text("owned fixture source")
            manifest.write_text(json.dumps(["source.go"]))
            self.assertEqual(len(check.source_hashes(root)), 1)
            for invalid in [["source.go", "source.go"], ["../source.go"], ["/source.go"]]:
                manifest.write_text(json.dumps(invalid))
                with self.assertRaises(AssertionError):
                    check.source_hashes(root)
            p = root / "profile"
            good = "mode: atomic\nfixture/http_transport.go:1.1,2.1 10 1\nfixture/http_ingress.go:1.1,2.1 10 1\nfixture/other.go:1.1,2.1 1 0\n"
            p.write_text(good)
            self.assertEqual(len(check.coverage(p)), 2)
            p.write_text(good.replace("10 1", "10 0"))
            with self.assertRaises(AssertionError):
                check.coverage(p)
            p.write_text("mode: atomic\n")
            with self.assertRaises(AssertionError):
                check.coverage(p)

    def test_memory_witness_closed_types_and_phase_boundaries(self):
        for case in check.INGRESS_CASES:
            base=mock_ingress(case);check.check_ingress_witness(base)
            mutations=[('target','https://wrong.invalid'),('peer_did',check.INGRESS_PEER),('source_version','4'),('source_service','https://wrong.invalid'),('tls_name','agents.example.com'),('input_sha256','0'*64),('written',True),('handler_calls',True),('authenticated_response',1),('verified_success',1),('offered',['h2','http/1.1']),('core_trace','OBSERVED'),('extra',1)]
            if case=='h2-only':mutations += [('handshake_error','i/o timeout'),('handshake_failed',False),('response_status',401),('calibration',{})]
            elif case=='positive-calibration':mutations += [('verified_payload_hex',''),('accepted_requests',0),('response_status',401),('response_error','authentication failed'),('handler_payloads_hex',[])]
            else:mutations += [('accepted_requests',1),('handler_calls',1),('response_error','i/o timeout'),('response_body_hex',''),('calibration',None)]
            for key,value in mutations:
                bad=json.loads(json.dumps(base));bad[key]=value
                with self.subTest(case=case,key=key),self.assertRaises((AssertionError,ValueError,TypeError,KeyError)):check.check_ingress_witness(bad)
        raw=check.unsigned_frame('bare-jsonrpc')
        self.assertIn(b'A2A-Version: 1.0\r\n',raw)
        self.assertNotIn(b'A2A-Version:',check.unsigned_frame('legacy-unsigned'))
        for raw in [b'',b'x',b'HTTP/1.1 200 OK\r\nContent-Length: 1\r\nContent-Length: 1\r\n\r\nx',b'HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nx']:
            with self.assertRaises((AssertionError,ValueError)):check.http_frame(raw)

    def test_memory_witness_fragment_attribution_and_duplicates(self):
        rows=good_rows();parsed=check.ingress_observations(rows);self.assertEqual(set(parsed),check.INGRESS_CASES)
        fragmented=[]
        for row in rows:
            if 'INGRESS_MEMORY_OBSERVATION ' in row.get('Output',''):
                for i in range(0,len(row['Output']),17):fragmented.append({**row,'Output':row['Output'][i:i+17]})
            else:fragmented.append(row)
        self.assertEqual(check.ingress_observations(fragmented),parsed)
        for mode in ('missing','duplicate','wrong-leaf','truncated','duplicate-json-key'):
            bad=json.loads(json.dumps(rows));n=next(i for i,r in enumerate(bad) if 'INGRESS_MEMORY_OBSERVATION ' in r.get('Output',''))
            if mode=='missing':bad.pop(n)
            elif mode=='duplicate':bad.insert(n,bad[n].copy())
            elif mode=='wrong-leaf':bad[n]['Test']='TestOther/positive-calibration'
            elif mode=='truncated':bad[n]['Output']=bad[n]['Output'].rstrip('\n')
            else:bad[n]['Output']=bad[n]['Output'].replace('{"case":','{"case":"duplicate","case":',1)
            with self.subTest(mode=mode),self.assertRaises((AssertionError,ValueError,TypeError,KeyError)):check.ingress_observations(bad)
        bad=json.loads(json.dumps(rows));n=next(i for i,r in enumerate(bad) if 'INGRESS_MEMORY_OBSERVATION ' in r.get('Output',''));bad[n]['Output']='a'*262145
        with self.assertRaises(AssertionError):check.ingress_observations(bad)

    def test_positive_raw_frames_joint_hash_and_role_mutations(self):
        modes=['content-type','header-did','header-version','digest','kid-header','recipient','role','created-bool','message-id','request-hash','context','session']
        for mode in modes:
            bad=mock_ingress('positive-calibration')
            response=mode in {'role','message-id','request-hash','context','session'}
            field='response_frame_hex' if response else 'input_hex'
            start,headers,body=check.http_frame(bytes.fromhex(bad[field]));envelope=json.loads(body)
            if mode=='content-type':headers['content-type']='text/plain'
            elif mode=='header-did':headers['x-sage-did']='did:sage:web:agents.example.com:other'
            elif mode=='header-version':headers['x-sage-version']='1.0'
            elif mode=='kid-header':headers['signature-input']=headers['signature-input'].replace('#sign','#other')
            elif mode=='recipient':envelope['recipient']='did:sage:web:agents.example.com:other'
            elif mode=='role':envelope['role']='initiator'
            elif mode=='created-bool':envelope['created']=True
            elif mode=='message-id':envelope['message_id']='00000000-0000-4000-8000-000000000009'
            elif mode=='request-hash':envelope['request_hash']=base64.urlsafe_b64encode(bytes(32)).rstrip(b'=').decode()
            elif mode=='context':envelope['context_id']='00000000-0000-4000-8000-000000000009'
            elif mode=='session':envelope['session_id']=base64.urlsafe_b64encode(bytes(16)).rstrip(b'=').decode()
            body=check.canonical(envelope);headers['content-length']=str(len(body));headers['content-digest']='sha-256=:'+base64.b64encode(hashlib.sha256(body).digest()).decode()+':'
            if mode=='digest':headers['content-digest']='sha-256=:'+base64.b64encode(bytes(32)).decode()+':'
            raw=(start+'\r\n'+''.join(k+': '+v+'\r\n' for k,v in headers.items())+'\r\n').encode()+body
            bad[field]=raw.hex()
            if response:bad['response_body_hex']=body.hex()
            else:bad['written']=len(raw);bad['input_sha256']=hashlib.sha256(raw).hexdigest()
            with self.subTest(mode=mode),self.assertRaises((AssertionError,ValueError,TypeError,KeyError)):check.check_ingress_witness(bad)

    def test_both_focused_and_full_require_actual_memory_witnesses(self):
        good=good_rows();bad=[r for r in good if 'INGRESS_MEMORY_OBSERVATION ' not in r.get('Output','')]
        for focused,full in [(bad,good),(good,bad)]:
            with self.assertRaises(AssertionError):check.check_tests(focused,full)
        result=check.check_tests(good,good)
        self.assertEqual(result['memory_ingress_controls'],5)
        self.assertEqual(set(result['ingress_memory_observations']),{'focused','full'})

    def test_cli_stage_contracts(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "verification/protected-http").mkdir(parents=True)
            (root / "verification/library-consumer").mkdir(parents=True)
            (root / "verification/protected-http/source-files.json").write_text(json.dumps(["source.go"]))
            (root / "source.go").write_text("source")
            expected = {"core": {"Version": "v1", "Sum": "h1:fixture", "GoModSum": "h1:mod"}}
            (root / "verification/library-consumer/did-dependencies.json").write_text(json.dumps(expected))
            out = root / "output"
            out.mkdir()
            previous = check.ROOT
            check.ROOT = root
            original_hashes = check.source_hashes
            check.source_hashes = lambda: original_hashes(root)
            try:
                check.main("sources", out)
                check.main("unchanged", out)
                (root / "source.go").write_text("changed")
                with self.assertRaises(AssertionError):
                    check.main("unchanged", out)
                (out / "modules.json").write_text(json.dumps({"Path": "core", **expected["core"]}))
                check.main("modules", out)
                rows = "\n".join(json.dumps(r) for r in good_rows())
                for name in ("tests.jsonl", "all-tests.jsonl"):
                    (out / name).write_text(rows)
                for name in ("focused.cover", "all.cover"):
                    (out / name).write_text("mode: atomic\nf/http_transport.go:1.1,2.1 10 1\nf/http_ingress.go:1.1,2.1 10 1\n")
                with contextlib.redirect_stdout(io.StringIO()):
                    check.main("tests", out)
                self.assertEqual(json.loads((out / "summary.json").read_text())["leaf_tests"], 135)
                with self.assertRaises(ValueError):
                    check.main("invalid", out)
            finally:
                check.ROOT = previous
                check.source_hashes = original_hashes


if __name__ == "__main__":
    unittest.main()
