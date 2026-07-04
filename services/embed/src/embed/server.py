from embed.embedder import LocalReranker
from kmap.v1 import embed_pb2, embed_pb2_grpc


class EmbedServicer(embed_pb2_grpc.EmbedServiceServicer):
    def __init__(self, backend, reranker=None) -> None:
        self._backend = backend
        self._reranker = reranker or LocalReranker()

    def Embed(self, request, context):
        vectors = self._backend.embed(list(request.texts))
        return embed_pb2.EmbedResponse(
            vectors=[embed_pb2.Embedding(values=vector) for vector in vectors]
        )

    def Rerank(self, request, context):
        scored = self._reranker.rerank(request.query, list(request.passages))
        return embed_pb2.RerankResponse(
            scores=[embed_pb2.RerankScore(index=index, score=score) for index, score in scored]
        )
