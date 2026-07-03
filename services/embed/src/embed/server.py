from embed import embedder
from kmap.v1 import embed_pb2, embed_pb2_grpc


class EmbedServicer(embed_pb2_grpc.EmbedServiceServicer):
    def __init__(self, backend) -> None:
        self._backend = backend

    def Embed(self, request, context):
        vectors = self._backend.embed(list(request.texts))
        return embed_pb2.EmbedResponse(
            vectors=[embed_pb2.Embedding(values=vector) for vector in vectors]
        )

    def Rerank(self, request, context):
        scored = embedder.rerank(request.query, list(request.passages))
        return embed_pb2.RerankResponse(
            scores=[embed_pb2.RerankScore(index=index, score=score) for index, score in scored]
        )
