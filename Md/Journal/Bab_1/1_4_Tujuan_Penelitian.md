# 1.4 Tujuan Penelitian

Tujuan utama penelitian ini adalah merancang dan mengimplementasikan prototipe backend workflow governance untuk monitoring proyek infrastruktur transnasional berbasis ODA.

Secara khusus, penelitian ini bertujuan untuk:

1. merancang model workflow berbasis Finite State Machine pada proses progress monitoring, ticket hambatan, dan approval pembayaran;

2. menerapkan organization-centric access control agar authorization mempertimbangkan user, organisasi, role, project, dan contract;

3. menerapkan SLA tracking untuk mengukur durasi proses serta audit trail untuk mencatat aktivitas penting;

4. menerapkan realtime event delivery berbasis outbox pattern dan WebSocket sebagai sinyal perubahan data;

5. mengevaluasi sistem melalui pengujian workflow, policy, SLA, audit/outbox, WebSocket, dan contract.
